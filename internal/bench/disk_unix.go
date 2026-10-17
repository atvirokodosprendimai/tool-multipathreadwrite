//go:build !windows

package bench

import "syscall"

// freeGiB is the space free to an unprivileged user on the volume holding path.
func freeGiB(path string) (float64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, err
	}
	return float64(st.Bavail) * float64(st.Bsize) / (1 << 30), nil
}
