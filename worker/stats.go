package worker

import (
	"log"

	"github.com/c9s/goprocinfo/linux"
)

// TODO add TaskCount
type Stats struct {
	CpuStats  *linux.CPUStat
	MemStats  *linux.MemInfo
	DiskStats *linux.Disk
	LoadStats *linux.LoadAvg
}

// -----cpu methods-----//
func (s *Stats) CpuUsage() float64 {
	idle := s.CpuStats.Idle + s.CpuStats.IOWait
	nonIdle := s.CpuStats.System + s.CpuStats.User + s.CpuStats.Nice + s.CpuStats.IRQ + s.CpuStats.SoftIRQ + s.CpuStats.Steal
	total := idle + nonIdle

	if total == 0 {
		return 0.00
	}

	return (float64(total) - float64(idle)) / float64(total)
}

// -----memory methods-----//
func (s *Stats) MemTotalKb() uint64 {
	return s.MemStats.MemTotal
}

func (s *Stats) MemAvailableKb() uint64 {
	return s.MemStats.MemAvailable
}

func (s *Stats) MemUsedKb() uint64 {
	return s.MemTotalKb() - s.MemAvailableKb()
}

func (s *Stats) MemUsedPercent() uint64 {
	return s.MemAvailableKb() / s.MemTotalKb()
}

// -----disk methods-----//
func (s *Stats) DiskTotal() uint64 {
	return s.DiskStats.All
}

func (s *Stats) DiskFree() uint64 {
	return s.DiskStats.Free
}

func (s *Stats) DiskUsed() uint64 {
	return s.DiskStats.Used
}

// --------------------------//

func GetStats() *Stats {
	return &Stats{
		CpuStats:  GetCpuStats(),
		MemStats:  GetMemInfo(),
		DiskStats: GetDiskInfo(),
		LoadStats: GetLoadAvg(),
	}
}

// -----helper functions-----//
func GetCpuStats() *linux.CPUStat {
	stats, err := linux.ReadStat("/proc/stat")

	if err != nil {
		log.Printf("error reading from /proc/stats: %v", err)
		return &linux.CPUStat{}
	}

	return &stats.CPUStatAll
}

func GetMemInfo() *linux.MemInfo {
	stats, err := linux.ReadMemInfo("/proc/meminfo")

	if err != nil {
		log.Printf("error reading from /proc/meminfo: %v", err)
		return &linux.MemInfo{}
	}

	return stats
}

func GetDiskInfo() *linux.Disk {
	stats, err := linux.ReadDisk("/")

	if err != nil {
		log.Printf("error reading from /: %v", err)
		return &linux.Disk{}
	}

	return stats
}

func GetLoadAvg() *linux.LoadAvg {
	stats, err := linux.ReadLoadAvg("/proc/loadavg")

	if err != nil {
		log.Printf("error reading from /proc/loadavg: %v", err)
		return &linux.LoadAvg{}
	}

	return stats
}
