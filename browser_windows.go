package main

import (
	"log"
	"os/exec"

	"golang.org/x/sys/windows/svc"
)

type windowsService struct{}

func (windowsService) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	done := make(chan error, 1)
	go func() { done <- run() }()
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case err := <-done:
			if err != nil {
				log.Printf("服务失败: %v", err)
				return false, 1
			}
			return false, 0
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				status <- request.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				close(serviceStop)
				if err := <-done; err != nil {
					log.Printf("服务停止失败: %v", err)
					return false, 1
				}
				return false, 0
			}
		}
	}
}

func runAsService() bool {
	service, err := svc.IsWindowsService()
	if err != nil || !service {
		return false
	}
	if err = svc.Run("NavHub", windowsService{}); err != nil {
		log.Printf("Windows 服务失败: %v", err)
	}
	return true
}

func openBrowser(url string) {
	service, e := svc.IsWindowsService()
	if e != nil || service {
		return
	}
	if e = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run(); e != nil {
		log.Print("浏览器未能自动打开，请手动访问服务地址")
	}
}
