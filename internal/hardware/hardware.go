package hardware

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/sirupsen/logrus"

	"patchmon-agent/pkg/models"
)

// Manager handles hardware information collection
type Manager struct {
	logger *logrus.Logger
}

// New creates a new hardware manager
func New(logger *logrus.Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// GetHardwareInfo collects hardware information
func (m *Manager) GetHardwareInfo() models.HardwareInfo {
	m.logger.Debug("Collecting hardware information...")

	info := models.HardwareInfo{
		CPUModel:     m.getCPUModel(),
		CPUCores:     m.getCPUCores(),
		RAMInstalled: m.getRAMSize(),
		SwapSize:     m.getSwapSize(),
		DiskDetails:  m.getDiskDetails(),
	}

	m.logger.Debugf("Hardware info collected - CPU: %s (%d cores), RAM: %.2fGB, Swap: %.2fGB, Disks: %d",
		info.CPUModel, info.CPUCores, info.RAMInstalled, info.SwapSize, len(info.DiskDetails))

	return info
}

// getCPUModel gets the CPU model name
func (m *Manager) getCPUModel() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := cpu.InfoWithContext(ctx)
	if err != nil {
		m.logger.Warnf("Failed to get CPU info: %v", err)
		return "Unknown"
	}

	if len(info) > 0 {
		return info[0].ModelName
	}

	return "Unknown"
}

// getCPUCores gets the number of CPU cores
func (m *Manager) getCPUCores() int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cores, err := cpu.CountsWithContext(ctx, true) // true for logical cores
	if err != nil {
		m.logger.Warnf("Failed to get CPU core count: %v", err)
		return 0
	}

	return cores
}

// getRAMSize gets the total RAM size in GB
func (m *Manager) getRAMSize() float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	memInfo, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		m.logger.Warnf("Failed to get memory info: %v", err)
		return 0
	}

	// Convert bytes to GB
	return float64(memInfo.Total) / (1024 * 1024 * 1024)
}

// getSwapSize gets the total swap size in GB
func (m *Manager) getSwapSize() float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	swapInfo, err := mem.SwapMemoryWithContext(ctx)
	if err != nil {
		m.logger.Warnf("Failed to get swap info: %v", err)
		return 0
	}

	// Convert bytes to GB
	return float64(swapInfo.Total) / (1024 * 1024 * 1024)
}

// getDiskDetails gets disk information
func (m *Manager) getDiskDetails() []models.DiskInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	partitions, err := disk.PartitionsWithContext(ctx, false) // false for physical devices only
	if err != nil {
		m.logger.Warnf("Failed to get disk partitions: %v", err)
		return []models.DiskInfo{}
	}

	var disks []models.DiskInfo

	for _, partition := range partitions {
		// Skip special filesystems
		if partition.Fstype == "tmpfs" || partition.Fstype == "devtmpfs" ||
			partition.Fstype == "proc" || partition.Fstype == "sysfs" ||
			partition.Fstype == "devpts" || partition.Fstype == "squashfs" {
			continue
		}

		usage, err := disk.UsageWithContext(ctx, partition.Mountpoint)
		if err != nil {
			m.logger.Warnf("Failed to get disk usage for %s: %v", partition.Mountpoint, err)
			continue
		}

		diskInfo := models.DiskInfo{
			Name: partition.Device,
			Size: fmt.Sprintf("%.2fGB (%.2fGB used, %.2fGB free, %.1f%% used)",
				float64(usage.Total)/(1024*1024*1024),
				float64(usage.Used)/(1024*1024*1024),
				float64(usage.Free)/(1024*1024*1024),
				usage.UsedPercent),
			MountPoint: partition.Mountpoint,
		}

		disks = append(disks, diskInfo)
	}

	return disks
}
