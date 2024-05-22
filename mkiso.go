package main

import (
	"io"
	"log"
	"os"

	diskfs "github.com/diskfs/go-diskfs"
	diskpkg "github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/diskfs/go-diskfs/partition/gpt"
)

func main() {
	if err := do(); err != nil {
		log.Panic(err)
	}
}

func do() error {
	diskImg := "out.iso"
	espSize := 2 << 20
	diskSize := espSize + 2<<20
	blkSize := diskfs.SectorSize(2048)
	partitionStart := 2048
	partitionSectors := espSize / int(blkSize)
	partitionEnd := partitionSectors - partitionStart + 1

	_ = os.Rename(diskImg, diskImg+".bak")

	// create a disk image
	disk, err := diskfs.Create(diskImg, int64(diskSize), diskfs.Raw, blkSize)
	if err != nil {
		return err
	}
	// create a partition table
	table := &gpt.Table{
		Partitions: []*gpt.Partition{
			{
				Start: uint64(partitionStart),
				End:   uint64(partitionEnd),
				Type:  gpt.EFISystemPartition,
				Name:  "EFI System",
			},
		},
	}
	// apply the partition table
	if err := disk.Partition(table); err != nil {
		return err
	}
	spec := diskpkg.FilesystemSpec{Partition: 0, FSType: filesystem.TypeISO9660}
	fs, err := disk.CreateFilesystem(spec)
	if err != nil {
		return err
	}

	// make our directories
	if err := fs.Mkdir("/boot/limine"); err != nil {
		return err
	}
	if err := fs.Mkdir("/EFI/BOOT"); err != nil {
		return err
	}

	// copy files
	// cf. limine's README for the required files
	for _, item := range []struct {
		src string
		dst string
	}{
		{"kernel/kernel.elf", "kernel.elf"},
		{"kernel/limine.cfg", "/boot/limine.cfg"},
		{"kernel/limine/limine-bios.sys", "/boot/limine-bios.sys"},
		{"kernel/limine/limine-bios-cd.bin", "/boot/limine-bios-cd.bin"},
		{"kernel/limine/limine-uefi-cd.bin", "/boot/limine-uefi-cd.bin"},
		{"kernel/limine/BOOTX64.EFI", "/EFI/BOOT/BOOTX64.EFI"},
		{"kernel/limine/BOOTIA32.EFI", "/EFI/BOOT/BOOTIA32.EFI"},
	} {
		dst, err := fs.OpenFile(item.dst, os.O_CREATE|os.O_RDWR)
		if err != nil {
			return err
		}
		src, err := os.Open(item.src)
		if err != nil {
			return err
		}
		_, err = io.Copy(dst, src)
		_ = src.Close()
		_ = dst.Close()
		if err != nil {
			return err
		}
	}

	options := iso9660.FinalizeOptions{
		VolumeIdentifier: "my-volume",
		RockRidge:        true,
		ElTorito: &iso9660.ElTorito{
			BootCatalog: "boot.cat",
			Entries: []*iso9660.ElToritoEntry{
				{
					Platform:  iso9660.BIOS,
					Emulation: iso9660.NoEmulation,
					BootFile:  "/boot/limine-bios-cd.bin",
					BootTable: true,
					LoadSize:  4,
				},
				{
					Platform:  iso9660.EFI,
					Emulation: iso9660.NoEmulation,
					BootFile:  "/boot/limine-uefi-cd.bin",
					BootTable: true,
				},
			},
		},
	}
	err = fs.(*iso9660.FileSystem).Finalize(options)

	return err
}
