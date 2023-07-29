package node

type Node struct {
	Name            string
	Ip              string
	Cores           int
	Memory          int // max memory
	MemoryAllocated int
	Disk            int // max disk space
	DiskAllocated   int
	Role            string
	TaskCount       int
}
