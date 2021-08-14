package  main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var fileCount = 0
var dirCount = 0
var foundCount = 0
var totalSize  int64 = 0
var detailMode = false
var targetFolder = ""
var keywords = ""

var (
	kernel32Dll    *syscall.LazyDLL  = syscall.NewLazyDLL("Kernel32.dll")
	setConsoleMode *syscall.LazyProc = kernel32Dll.NewProc("SetConsoleMode")
)

func EnableVirtualTerminalProcessing(stream syscall.Handle, enable bool) error {
	const ENABLE_VIRTUAL_TERMINAL_PROCESSING uint32 = 0x4

	var mode uint32
	err := syscall.GetConsoleMode(syscall.Stdout, &mode)
	if err != nil {
		return err
	}

	if enable {
		mode |= ENABLE_VIRTUAL_TERMINAL_PROCESSING
	} else {
		mode &^= ENABLE_VIRTUAL_TERMINAL_PROCESSING
	}

	ret, _, err := setConsoleMode.Call(uintptr(stream), uintptr(mode))
	if ret == 0 {
		return err
	}

	return nil
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// folderExist check if a folder exists
func folderExists(folderName string) bool {
	info, err := os.Stat(folderName)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func GetFilesAndDirs(dirPath string) (files []string,dirs []string,err error) {
	dir,err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil,nil,err
	}

	symbol := string(os.PathSeparator)

	for _,fi := range dir {
		if fi.IsDir() {
			dirCount++
			dirs := append(dirs,dirPath + symbol + fi.Name())
			s := strings.Join(dirs,"")
			if len(s) >= 80 {
				s = s[:78]
				s = s + "..."
			}
			//fmt.Fprintf(os.Stdout,"\033[0;0H")
			//fmt.Fprintf(os.Stdout, "Folder %d: %s", dirCount, dirs)
			//fmt.Fprintf(os.Stdout,"\r\x1b[K")
			//\x1ba
			fmt.Fprintf(os.Stdout,"\r \r")
			fmt.Fprintf(os.Stdout,"\x1b[A")
			fmt.Fprintf(os.Stdout,"\x1b[2K")
			fmt.Fprintf(os.Stdout, "Folder %d: %s      【Files:%d】\n", dirCount, s,fileCount)
			GetFilesAndDirs(dirPath + symbol + fi.Name())
		} else {
			fileCount++
			totalSize += fi.Size()
			var fname = fi.Name()
			var f = dirPath+symbol + fname
			if detailMode {
				if len(keywords)>0 {
					var matched, err = filepath.Match(keywords, fname)
					if err != nil {
						fmt.Println(err)
					} else {
						if matched {
							fmt.Printf("  File %d: %s\n", fileCount, f)
							foundCount++
							var newFile = targetFolder + symbol + fi.Name()
							for fileExists(newFile) {
								fname = "_" + fname
								newFile = dirPath+symbol + fname
							}

							if len(targetFolder) > 0 {
								err := CopyFile(f, newFile)
								if err != nil {
									fmt.Printf("CopyFile failed %q\n", err)
								} else {
									fmt.Printf("CopyFile %s succeeded\n", fi.Name())
								}
							}

						}
					}


				} else {
					fmt.Printf("  File %d: %s\n", fileCount, f)
				}
			}
			files = append(files,f)
			}
		}

	return files,dirs,nil
}

func ByteCountDecimal(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

func main() {
	start := time.Now()

	EnableVirtualTerminalProcessing(syscall.Stdout, true)
	fmt.Println("File system counter & search utility v1.0. \nDesigned by Hu Gang (https://github.com/ihugang)")
	fmt.Println("Usage: find2 [folder] [-detail] -search:<keywords> -output:<target folder>")
	fmt.Println("          -detail                  -> display file")
	fmt.Println("          -search: <keywords>      -> search files that name contains keywords")
	fmt.Println("          -output: <target folder> -> copy found files to the folder")
	fmt.Println("")
	var initDir = "./"
	for idx,args := range os.Args {
		//fmt.Println("Params " + strconv.Itoa(idx) + ":" + args)
		if idx == 1 {
			initDir = args
		}

		if idx >= 1 {
			if strings.Contains(args, "-detail") || strings.Contains(args, "/detail")   {
				detailMode = true
			}

			if strings.Contains(args, "-search") || strings.Contains(args, "/search")   {
				var arr = strings.Split(args,":")
				if len(arr) == 2 {
					keywords =  strings.Trim(arr[1],"")
					fmt.Println("Keywords: " + keywords)
					detailMode = true
				}
			}

			if strings.Contains(args, "-output") || strings.Contains(args, "/output")   {
				var charIndex = strings.Index(args,":")
				if charIndex > 0 {
					targetFolder = args[charIndex+1:]
					if !folderExists(targetFolder) {
						os.Mkdir(targetFolder,0755)
					}
					fmt.Println("Copy to : " + targetFolder)
				}
			}
		}
	}
	fmt.Println("\nnow discover [" + initDir + "]")
	_, _, _ = GetFilesAndDirs(initDir)
	//fmt.Fprintf(os.Stdout,"\r\x1b[K")
	fmt.Fprintf(os.Stdout,"\x1b[A")
	fmt.Fprintf(os.Stdout,"\x1b[2K")
	fmt.Fprintf(os.Stdout,"\r \r")

	fmt.Println("Result: totally " +  strconv.Itoa(dirCount) + " folders, " + strconv.Itoa(fileCount) + " files, " + ByteCountDecimal(totalSize) + " bytes")
	fmt.Println("        found " +  strconv.Itoa(foundCount) + " files. ")
	elapsed := time.Since(start)
	fmt.Fprintf(os.Stdout,"Binomial took %s\n", elapsed)
}
