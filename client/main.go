package main

import (
	"archive/zip"
	"embed"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/elitah/fast-io"
	"github.com/elitah/utils/wait"

	"github.com/xtaci/smux"
)

//go:embed jsmpeg-vnc-v0.2.zip
var fs embed.FS

var (
	//
	hKernel32 uintptr
	//
	hProcessExitGroup ProcessExitGroup
	//
	fLock *os.File
)

func init() {
	//
	if h, err := syscall.LoadLibrary("kernel32.dll"); nil == err {
		//
		atomic.StoreUintptr(&hKernel32, uintptr(h))
	} else {
		//
		fmt.Println(err)
	}
}

func loadKernel32() (syscall.Handle, error) {
	//
	if p := atomic.LoadUintptr(&hKernel32); 0 != p {
		//
		return syscall.Handle(p), nil
	} else {
		//
		return syscall.Handle(0), fmt.Errorf("no such library")
	}
}

func lockFile(fd uintptr) (ret bool) {
	//
	if h, err := loadKernel32(); nil == err {
		//
		if addr, err := syscall.GetProcAddress(h, "LockFile"); nil == err {
			//
			r0, _, _ := syscall.Syscall6(addr, 5, fd, 0, 0, 0, 1, 0)
			//
			ret = 0 != int(r0)
		}
	}
	//
	return ret
}

func testFileIsLocked(path string) (ret bool) {
	//
	if f, err := os.Open(path); nil == err {
		//
		ret = !lockFile(f.Fd())
		//
		f.Close()
	}
	//
	return
}

type process struct {
	Pid    int
	Handle uintptr
}

type ProcessExitGroup windows.Handle

func NewProcessExitGroup() (ProcessExitGroup, error) {
	//
	if handle, err := windows.CreateJobObject(nil, nil); nil == err {
		//
		info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
			BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
				LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
			},
		}
		//
		if _, err := windows.SetInformationJobObject(
			handle,
			windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)),
			uint32(unsafe.Sizeof(info))); err != nil {
			return 0, err
		}
		//
		return ProcessExitGroup(handle), nil
	} else {
		//
		return 0, err
	}
}

func (g ProcessExitGroup) Dispose() error {
	//
	return windows.CloseHandle(windows.Handle(g))
}

func (g ProcessExitGroup) AddProcess(p *os.Process) error {
	//
	return windows.AssignProcessToJobObject(
		windows.Handle(g),
		windows.Handle((*process)(unsafe.Pointer(p)).Handle),
	)
}

type readAt struct {
	io.ReadSeekCloser
}

func (this *readAt) ReadAt(p []byte, off int64) (int, error) {
	//
	if _, err := this.Seek(off, io.SeekStart); nil == err {
		//
		return this.Read(p)
	} else {
		//
		return 0, err
	}
}

func extract() (dir string) {
	//
	if f, err := fs.Open("jsmpeg-vnc-v0.2.zip"); nil == err {
		//
		if info, err := f.Stat(); nil == err {
			//
			if _f, ok := f.(io.ReadSeekCloser); ok {
				//
				if zr, err := zip.NewReader(&readAt{_f}, info.Size()); nil == err {
					//
					dir = os.TempDir()
					//
					if list, err := filepath.Glob(fmt.Sprintf("%s/webvnc_*", dir)); nil == err {
						//
						for _, item := range list {
							//
							if !testFileIsLocked(
								filepath.Join(
									item,
									"jsmpeg-vnc.exe",
								),
							) {
								//
								os.RemoveAll(item)
							}
						}
					}
					//
					dir = fmt.Sprintf("%s/webvnc_%d", dir, time.Now().UnixNano())
					//
					os.Mkdir(dir, 0755)
					//
					for _, item := range zr.File {
						//
						if 0x0 != item.ExternalAttrs&0x16 {
							//
							os.Mkdir(filepath.Join(dir, item.Name), 0755)
						} else if src, err := item.Open(); nil == err {
							//
							if dst, err := os.OpenFile(
								//
								filepath.Join(
									dir,
									item.Name,
								),
								os.O_RDWR|os.O_CREATE|os.O_TRUNC,
								0644,
							); nil == err {
								//
								fast_io.Copy(dst, src)
								//
								dst.Close()
							} else {
								//
								fmt.Println(err)
							}
							//
							src.Close()
						} else {
							//
							fmt.Println(err)
						}
					}
					//
					if f, err := os.Open(filepath.Join(
						dir,
						"jsmpeg-vnc.exe",
					)); nil == err {
						//
						if lockFile(f.Fd()) {
							//
							fLock = f
						} else {
							//
							f.Close()
						}
					}
				} else {
					//
					fmt.Println(err)
				}
			}
		} else {
			//
			fmt.Println(err)
		}
	} else {
		//
		fmt.Println(err)
	}
	//
	return
}

func calcPort() int {
	//
	if l, err := net.ListenTCP("tcp4", &net.TCPAddr{}); nil == err {
		//
		defer l.Close()
		//
		if addr, ok := l.Addr().(*net.TCPAddr); ok {
			//
			return addr.Port
		}
	}
	//
	return -1
}

func startService(root string) (int, chan struct{}) {
	//
	if port := calcPort(); 0 < port {
		//
		var f [4]*os.File
		//
		f[0], f[1], _ = os.Pipe()
		//
		f[2], f[3], _ = os.Pipe()
		//
		if p, err := os.StartProcess(
			filepath.Join(
				root,
				"jsmpeg-vnc.exe",
			),
			[]string{
				"jsmpeg-vnc.exe",
				"-b", "1000",
				"-s", "640x480",
				"-f", "5",
				"-p", fmt.Sprint(port),
				"desktop",
			},
			&os.ProcAttr{
				Dir:   root,
				Files: []*os.File{f[0], f[3], f[3]},
				Sys:   &syscall.SysProcAttr{HideWindow: true},
			},
		); nil == err {
			//
			ch := make(chan struct{})
			//
			if 0 != hProcessExitGroup {
				//
				if err := hProcessExitGroup.AddProcess(p); nil != err {
					//
					fmt.Println(err)
				}
			}
			//
			go func() {
				//
				var buffer [1024]byte
				//
				for {
					//
					if _, err := f[2].Read(buffer[:]); nil != err {
						//
						break
					}
				}
			}()
			//
			go func() {
				//
				<-ch
				//
				p.Signal(os.Interrupt)
			}()
			//
			go func() {
				//
				p.Wait()
				//
				for _, item := range f {
					//
					item.Close()
				}
				//
				select {
				case <-ch:
				default:
					//
					close(ch)
				}
			}()
			//
			return port, ch
		} else {
			//
			for _, item := range f {
				//
				item.Close()
			}
			//
			fmt.Println(err)
		}
	}
	//
	return -1, nil
}

func main() {
	//
	var fLockMain *os.File
	//
	var ch chan struct{}
	//
	var dir string
	//
	var server string
	//
	flag.StringVar(&server, "s", "", "your server address")
	//
	flag.Parse()
	//
	if "" == server {
		//
		return
	}
	//
	if f, err := os.Open(os.Args[0]); nil == err {
		//
		if lockFile(f.Fd()) {
			//
			fLockMain = f
		} else {
			//
			return
		}
	} else {
		//
		return
	}
	//
	dir = extract()
	//
	if "" == dir {
		//
		return
	}
	//
	defer func(dir string) {
		//
		if nil != fLock {
			//
			fLock.Close()
		}
		//
		if nil != fLockMain {
			//
			fLockMain.Close()
		}
		//
		os.RemoveAll(dir)
	}(dir)
	//
	if g, err := NewProcessExitGroup(); nil == err {
		//
		defer g.Dispose()
		//
		hProcessExitGroup = g
	}
	//
	go func(dir, server string) {
		//
		for {
			//
			if addr, err := net.ResolveTCPAddr("tcp4", server); nil == err {
				//
				if conn, err := net.DialTCP("tcp4", nil, addr); nil == err {
					//
					var port int
					//
					if session, err := smux.Server(conn, smux.DefaultConfig()); nil == err {
						//
						for {
							//
							if stream, err := session.AcceptStream(); nil == err {
								//
								if 0 < port {
									//
									select {
									case <-ch:
										//
										port = -1
									default:
									}
								}
								//
								if 0 >= port {
									//
									port, ch = startService(dir)
								}
								//
								if 0 < port {
									//
									go func(remote net.Conn, port int) {
										//
										if local, err := net.DialTCP("tcp4", nil, &net.TCPAddr{
											IP:   net.IPv4(127, 0, 0, 1),
											Port: port,
										}); nil == err {
											//
											fast_io.FastCopy(local, remote)
											//
											local.Close()
										}
										//
										remote.Close()
									}(stream, port)
								} else {
									//
									stream.Close()
								}
							} else {
								//
								break
							}
						}
					} else {
						//
						fmt.Println(err)
					}
					//
					conn.Close()
				} else {
					//
					fmt.Println(err)
				}
			} else {
				//
				fmt.Println(err)
			}
			//
			time.Sleep(3 * time.Second)
		}
	}(dir, server)
	//
	if err := wait.Signal(
		wait.WithSignal(os.Interrupt),
	); nil != err {
		//
		fmt.Println(err)
	}
}
