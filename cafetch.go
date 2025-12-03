package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// el type SystemInfo guarda toda la información del sistema
type SystemInfo struct {
	OS, Kernel, Arch, Host, User, Shell, Term, CPU, Uptime string
	MemUsed, MemTotal, DiskUsed, DiskTotal                 int
}

func main() {
	info := getSystemInfo()
	printInfo(info)
}

// la func getSystemInfo recolecta toda la información del sistema
func getSystemInfo() SystemInfo {
	info := SystemInfo{
		OS:     getOS(),
		Kernel: runCmd("uname", "-r"),
		Arch:   runtime.GOARCH,
		Host:   getEnvOrDefault("HOSTNAME", "N/A"),
		User:   getEnvOrDefault("USER", "N/A"),
		Shell:  getEnvOrDefault("SHELL", "N/A"),
		Term:   getEnvOrDefault("TERM", "N/A"),
		CPU:    getCPU(),
		Uptime: getUptime(),
	}

	// Memoria
	info.MemTotal, info.MemUsed = getMemory()

	// Disco
	info.DiskTotal, info.DiskUsed = getDisk("/")

	return info
}

// runCmd ejecuta un comando y devuelve su salida
func runCmd(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "N/A"
	}
	return strings.TrimSpace(string(out))
}

// getEnvOrDefault obtiene una variable de entorno o devuelve un valor por defecto
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getOS obtiene el nombre del sistema operativo
func getOS() string {
	// Intenta leer /etc/os-release primero
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return runtime.GOOS
	}
	defer file.Close()

	// Busca la línea PRETTY_NAME
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
		}
	}
	return runtime.GOOS
}

// getCPU obtiene el modelo de CPU
func getCPU() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "N/A"
	}
	defer file.Close()

	// Busca la línea "model name"
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "N/A"
}

// getUptime calcula el tiempo que lleva encendido el sistema
func getUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "N/A"
	}

	// Parsea los segundos desde /proc/uptime
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "N/A"
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "N/A"
	}

	// Convierte a días, horas y minutos
	s := int(seconds)
	days := s / 86400
	hours := (s % 86400) / 3600
	minutes := (s % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// getMemory obtiene la memoria total y usada en MB
func getMemory() (total, used int) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var memTotal, memAvail int

	// Lee las líneas de /proc/meminfo
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Extrae los valores en kilobytes
		val, _ := strconv.Atoi(fields[1])

		if strings.HasPrefix(line, "MemTotal:") {
			memTotal = val
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			memAvail = val
		}

		// Si esta el valor de MemTotal y MemAvailable ya no es necesario seguir leyendo
		if memTotal > 0 && memAvail > 0 {
			break
		}
	}

	// Convierte KB a MB
	total = memTotal / 1024
	used = total - (memAvail / 1024)
	return
}

// getDisk obtiene el espacio total y usado del disco en GB
func getDisk(path string) (total, used int) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0
	}

	// Calcula el espacio total y libre
	totalBytes := stat.Blocks * uint64(stat.Bsize)
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes

	// Convierte a GB
	gb := float64(1024 * 1024 * 1024)
	total = int(float64(totalBytes) / gb)
	used = int(float64(usedBytes) / gb)
	return
}

// printInfo imprime toda la información con formato bonito
func printInfo(info SystemInfo) {
	// Colores ANSI
	c := map[string]string{
		"reset":   "\033[0m",
		"bold":    "\033[1m",
		"cyan":    "\033[36m",
		"magenta": "\033[35m",
		"yellow":  "\033[33m",
		"green":   "\033[32m",
	}

	// Logo en formato ASCII de una taza de cafe :D
	logo := []string{
		c["cyan"] + "     ( (  " + c["reset"],
		c["cyan"] + "      ) ) " + c["reset"],
		c["yellow"] + "  ........ " + c["reset"],
		c["yellow"] + "  |      |]" + c["reset"],
		c["yellow"] + "  |      | " + c["reset"],
		c["yellow"] + "   ======  " + c["reset"],
	}

	// Calcula porcentajes
	memPercent := 0.0
	if info.MemTotal > 0 {
		memPercent = float64(info.MemUsed) / float64(info.MemTotal) * 100
	}
	diskPercent := 0.0
	if info.DiskTotal > 0 {
		diskPercent = float64(info.DiskUsed) / float64(info.DiskTotal) * 100
	}

	// Información del sistema
	data := []string{
		c["bold"] + info.User + "@" + info.Host + c["reset"],
		c["cyan"] + "cafetch" + c["reset"] + " (Go " + runtime.Version() + ")",
		"",
		c["yellow"] + "OS:     " + c["reset"] + info.OS,
		c["yellow"] + "Kernel: " + c["reset"] + info.Kernel,
		c["yellow"] + "Arch:   " + c["reset"] + info.Arch,
		c["yellow"] + "Uptime: " + c["reset"] + info.Uptime,
		"",
		c["green"] + "CPU:  " + c["reset"] + info.CPU,
		fmt.Sprintf(c["green"]+"Mem:  "+c["reset"]+"%dMB / %dMB (%.1f%%)", info.MemUsed, info.MemTotal, memPercent),
		fmt.Sprintf(c["green"]+"Disk: "+c["reset"]+"%dGB / %dGB (%.1f%%)", info.DiskUsed, info.DiskTotal, diskPercent),
		"",
		c["magenta"] + "Shell: " + c["reset"] + info.Shell,
		c["magenta"] + "Term:  " + c["reset"] + info.Term,
		c["magenta"] + "Time:  " + c["reset"] + time.Now().Format("2006-01-02 15:04:05"),
	}

	// Imprime logo e info lado a lado
	maxLines := len(logo)
	if len(data) > maxLines {
		maxLines = len(data)
	}

	for i := 0; i < maxLines; i++ {
		// Obtiene línea del logo
		logoLine := ""
		if i < len(logo) {
			logoLine = logo[i]
		}

		// Obtiene línea de datos
		dataLine := ""
		if i < len(data) {
			dataLine = data[i]
		}

		// Imprime las 2 con espaciado
		fmt.Printf("  %-20s  %s\n", logoLine, dataLine)
	}
}
