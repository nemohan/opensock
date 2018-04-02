package utility

import(
	"strings"
	"compress/gzip"
	"os"
	"io"
)

const blockSize = 16 * 4096
//ListFiles list all the files in the specified dir which begin with prefix and end with suffix 
func ListFiles(dir string, log *LogContext, prefix, suffix string) ([]string, error) {
	curDir, err := os.Open(dir)
	if err != nil {
		log.LogWarn("%v", err)
		return nil, err
	}
	defer curDir.Close()
	files, err := curDir.Readdir(0)
	if err != nil {
		log.LogWarn("%v", err)
		return nil, err
	}

	//target := make([]LogFile, 0)
	target := make([]string, 0)
	for _, f := range files {
		name := f.Name()
		if f.IsDir() || !strings.HasSuffix(name, suffix) || !strings.HasPrefix(name, prefix){
			log.LogDebug("file:%s prefix:%s passed", name, prefix)
			continue
		}
		
		log.LogDebug("target file:%s", name)
		//strDate := strings.TrimPrefix(strings.TrimSuffix(name, suffix), prefix)
		//date, err := strconv.Atoi(strDate)
		if err != nil{
			log.LogInfo("%v", err)
		}
		//target = append(target, LogFile{name: name, date: uint64(date)})
		target = append(target, name)
	}

	return target, nil
}

//Zip compress the file which is located in path "path" and store it in backupDir.
//and remove the file when compress done
func Zip(path, file, backupDir string, log *LogContext){
	fileName := path + file
	name := backupDir + file + ".gz"
	dstFile, err := os.Create(name)
	if err != nil{
		log.LogInfo("%v", err)
		return
	}
	defer dstFile.Close()

	f, err := os.Open(fileName)
	if err != nil{
		log.LogInfo("%v", err)
		return
	}
	defer f.Close()

	writer := gzip.NewWriter(dstFile)
	buf := make([]byte, blockSize)
	for{
		n, err := f.Read(buf)
		if err == io.EOF || n == 0{
			break
		}
		writer.Write(buf[:n])
	}
	writer.Close()
	os.Remove(fileName)
	log.LogInfo("done: %s  zip file:%s", file, name)
}