package mtproto

import (
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

func (m *MTProto) GetPhoto(volumeID, secret int64, localID, offset int32, flag bool) (TL, error) {
	resp := make(chan response, 1)

	fileLocation := TL_inputFileLocation{volumeID, localID, secret}
	if flag {
		m.queueSend <- packetToSend{
			msg:  TL_upload_getFile{fileLocation, offset * 1024 * 100, int32(100 * 1024)},
			resp: resp,
		}
	} else {
		m.queueSend <- packetToSend{
			msg:  TL_upload_getFile{fileLocation, offset * 0, int32(0)},
			resp: resp,
		}
	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(30 * time.Second)
		timeout <- true
	}()

	select {
	case x := <-resp:
		if x.err != nil {
			return nil, x.err
		}
		return x.data, nil
	case <-timeout:
		log.Println("time out on response")
		return nil, ErrTelegramTimeOut
	}

}

func (m *MTProto) GetFile(id, hash int64, offset int32) (TL, error) {
	resp := make(chan response, 1)

	// fileLocation := TL_inputDocumentFileLocation{id, hash, version}
	fileLocation := TL_inputEncryptedFileLocation{id, hash}
	m.queueSend <- packetToSend{
		msg:  TL_upload_getFile{fileLocation, offset * 1024 * 100, int32(100 * 1024)},
		resp: resp,
	}

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(1 * time.Minute)
		timeout <- true
	}()

	select {
	case x := <-resp:
		if x.err != nil {
			return nil, x.err
		}
		return x.data, nil
	case <-timeout:
		log.Println("time out on response")
		return nil, ErrTelegramTimeOut
	}

}

func (m *MTProto) DownloadImage(volumeID, secret, localID, size, path string) ([]byte, error) {
	offset := 0
	status := true

	vol, _ := strconv.ParseInt(volumeID, 10, 64)
	secret64, _ := strconv.ParseInt(secret, 10, 64)
	loc, _ := strconv.Atoi(localID)
	size64, _ := strconv.Atoi(size)
	var response string

	// filename := fmt.Sprintf("%s/%s.jpg", path, localID)
	// f, _ := os.Create(filename)
	// defer f.Close()
	length := 0
	for status {
		if size64 > 0 {
			// stat, _ := f.Stat()
			// if stat.Size() < int64(size64) {
			if length < size64 {
				x, err := m.GetPhoto(vol, secret64, int32(loc), int32(offset), true)
				if err != nil {
					return nil, err
				}

				list, _ := x.(TL_upload_file)
				// f.Write(list.Bytes)
				response += string(list.Bytes)
				length += len(response)

				offset++
			} else {
				status = false
			}

		} else {
			x, err := m.GetPhoto(vol, secret64, int32(loc), int32(offset), false)
			if err != nil {
				return nil, err
			}

			list, _ := x.(TL_upload_file)
			// f.Write(list.Bytes)
			response = string(list.Bytes)
			length = len(response)

			status = false
		}

	}
	log.Printf("Get: https://cdn.zelkaa.com/telegram/images/%s/%s/%s", volumeID, localID, secret)
	return []byte(response), nil
}

// func (m *MTProto) DownloadVideo(id, hash, size, path string) error {
// 	status := true
// 	offset := 0

// 	size64, _ := strconv.Atoi(size)
// 	access_hash, _ := strconv.ParseInt(hash, 10, 64)
// 	ID, _ := strconv.ParseInt(id, 10, 64)

// 	filename := fmt.Sprintf("%s/%s.mp4", path, id)
// 	f, _ := os.Create(filename)
// 	defer f.Close()

// 	for status {
// 		stat, _ := f.Stat()
// 		if stat.Size() < int64(size64) {
// 			x, err := m.GetFile(ID, access_hash, int32(offset))
// 			if err != nil {
// 				return err
// 			}

// 			list, _ := x.(TL_upload_file)
// 			f.Write(list.Bytes)

// 			offset++
// 		} else {
// 			status = false
// 		}

// 	}
// 	return nil
// }

func (m *MTProto) DownloadVideo(id, hash, size, path string) ([]byte, error) {
	status := true
	offset := 0
	var video string

	size64, _ := strconv.Atoi(size)
	accessHash, _ := strconv.ParseInt(hash, 10, 64)
	ID, _ := strconv.ParseInt(id, 10, 64)

	// filename := fmt.Sprintf("%s.mp4", path)
	// f, _ := os.Create(filename)
	// defer f.Close()
	log.Printf("starting download file %s", path)
	for status {
		if offset*1024*100 < size64 {

			x, err := m.GetFile(ID, accessHash, int32(offset))
			if err != nil {
				log.Println(err.Error())
				return nil, err
			}

			list, _ := x.(TL_upload_file)
			video += string(list.Bytes)
			offset++
			log.Println(offset)
		} else {
			status = false
		}
	}
	if len(video) != 0 {
		err := ioutil.WriteFile(path, []byte(video), 0644)
		if err != nil {
			log.Printf("error occurred download file %s", path)
			return nil, err
		}
		log.Printf("finished download file %s", path)
	}
	return []byte(video), nil
}

func (m *MTProto) SaveImage(path string, content []byte) {
	f, _ := os.Create(path)
	defer f.Close()
	f.Write(content)
}
