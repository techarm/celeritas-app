package handlers

import (
	"fmt"
	"io"
	"myapp/data"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/techarm/celeritas"
	"github.com/techarm/celeritas/filesystems"
	"github.com/techarm/celeritas/filesystems/miniofs"
	"github.com/techarm/celeritas/filesystems/s3fs"
	"github.com/techarm/celeritas/filesystems/sftpfs"
	"github.com/techarm/celeritas/filesystems/webdavfs"
)

type Handlers struct {
	App    *celeritas.Celeritas
	Models data.Models
}

func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	defer h.App.LoadTime(time.Now())
	err := h.render(w, r, "home", nil, nil)

	if err != nil {
		h.App.ErrorLog.Println("error rendering:", err)
	}
}

func (h *Handlers) CeleritasUpload(w http.ResponseWriter, r *http.Request) {
	err := h.render(w, r, "celeritas-upload", nil, nil)
	if err != nil {
		h.App.ErrorLog.Println(err)
	}
}

func (h *Handlers) PostCeleritasUpload(w http.ResponseWriter, r *http.Request) {
	err := h.App.UploadFile(r, "", "formFile", &h.App.Minio)
	if err != nil {
		h.App.ErrorLog.Println(err)
		h.App.Session.Put(r.Context(), "error", err.Error())
	} else {
		h.App.Session.Put(r.Context(), "flash", "Uploaded!")
	}
	http.Redirect(w, r, "/upload", http.StatusSeeOther)
}

func (h *Handlers) GoPage(w http.ResponseWriter, r *http.Request) {
	err := h.renderGoPage(w, r, "home", nil)
	if err != nil {
		h.App.ErrorLog.Println("error rendering:", err)
	}
}

func (h *Handlers) JetPage(w http.ResponseWriter, r *http.Request) {
	err := h.renderJetPage(w, r, "jet-template", nil, nil)
	if err != nil {
		h.App.ErrorLog.Println("error rendering:", err)
	}
}

func (h *Handlers) SessionPage(w http.ResponseWriter, r *http.Request) {
	myData := "bar"
	h.sessionPut(r.Context(), "foo", myData)

	myValue := h.sessionGet(r.Context(), "foo")

	vars := make(jet.VarMap)
	vars.Set("foo", myValue)

	err := h.renderJetPage(w, r, "sessions", vars, nil)
	if err != nil {
		h.App.ErrorLog.Println("error rendering:", err)
	}
}

func (h *Handlers) JSON(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ID      int64    `json:"id"`
		Name    string   `json:"name"`
		Hobbies []string `json:"hobbies"`
	}

	payload.ID = 10
	payload.Name = "Jack Jones"
	payload.Hobbies = []string{"karate", "running", "programming"}

	err := h.App.WriteJSON(w, http.StatusOK, payload)
	if err != nil {
		h.App.ErrorLog.Println(err)
	}
}

func (h *Handlers) XML(w http.ResponseWriter, r *http.Request) {
	type Payload struct {
		ID      int64    `xml:"id"`
		Name    string   `xml:"name"`
		Hobbies []string `xml:"hobbies>hobby"`
	}

	var payload Payload
	payload.ID = 11
	payload.Name = "Jack Jones"
	payload.Hobbies = []string{"karate", "running", "programming"}

	err := h.App.WriteXML(w, http.StatusOK, payload)
	if err != nil {
		h.App.ErrorLog.Println(err)
	}
}

func (h *Handlers) DownloadFile(w http.ResponseWriter, r *http.Request) {
	h.App.DownloadFile(w, r, "./public/images", "celeritas.jpg")
}

func (h *Handlers) TestCrypto(w http.ResponseWriter, r *http.Request) {
	plainText := "Hello, world"
	fmt.Fprintf(w, "Uncencrypted: "+plainText+"\n")

	encrypted, err := h.encrypt(plainText)
	if err != nil {
		h.App.ErrorLog.Println(err)
		h.App.ErrorInternalServerError(w, r)
		return
	}

	fmt.Fprintf(w, "Encrypted: "+encrypted+"\n")

	decrypted, err := h.decrypt(encrypted)
	if err != nil {
		h.App.ErrorLog.Println(err)
		h.App.ErrorInternalServerError(w, r)
		return
	}

	fmt.Fprintf(w, "Decrypted: "+decrypted+"\n")
}

func (h *Handlers) ListFS(w http.ResponseWriter, r *http.Request) {
	var fs filesystems.FS
	var list []filesystems.Listing

	fsType := ""
	if r.URL.Query().Get("fs-type") != "" {
		fsType = r.URL.Query().Get("fs-type")
	}

	curPath := ""
	if r.URL.Query().Get("curPath") != "" {
		curPath = r.URL.Query().Get("curPath")
		curPath, _ = url.QueryUnescape(curPath)
	}

	if fsType != "" {
		switch fsType {
		case "MINIO":
			f := h.App.FileSystems["MINIO"].(miniofs.Minio)
			fs = &f
			fsType = "MINIO"
		case "SFTP":
			f := h.App.FileSystems["SFTP"].(sftpfs.SFTP)
			fs = &f
			fsType = "SFTP"
		case "WEBDAV":
			f := h.App.FileSystems["WEBDAV"].(webdavfs.WebDAV)
			fs = &f
			fsType = "WEBDAV"
		case "S3":
			f := h.App.FileSystems["S3"].(s3fs.S3)
			fs = &f
			fsType = "S3"
		}

		fmt.Println(curPath)
		l, err := fs.List(curPath)
		if err != nil {
			h.App.ErrorLog.Println(err)
			return
		}

		list = l
	}

	vars := make(jet.VarMap)
	vars.Set("list", list)
	vars.Set("fs_type", fsType)
	vars.Set("curPath", curPath)

	err := h.render(w, r, "list-fs", vars, nil)
	if err != nil {
		h.App.ErrorLog.Println(err)
		return
	}
}

func (h *Handlers) UploadToFS(w http.ResponseWriter, r *http.Request) {
	fsType := r.URL.Query().Get("type")

	vars := make(jet.VarMap)
	vars.Set("fs_type", fsType)

	err := h.render(w, r, "upload", vars, nil)
	if err != nil {
		h.App.ErrorLog.Println(err)
		return
	}
}

func (h *Handlers) PostUploadToFS(w http.ResponseWriter, r *http.Request) {
	fileName, err := getFileToUpload(r, "formFile")
	fmt.Println(fileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	uploadType := r.Form.Get("upload-type")
	switch uploadType {
	case "MINIO":
		fs := h.App.FileSystems["MINIO"].(miniofs.Minio)
		err = fs.Put(fileName, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "SFTP":
		fs := h.App.FileSystems["SFTP"].(sftpfs.SFTP)
		err = fs.Put(fileName, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "WEBDAV":
		fs := h.App.FileSystems["WEBDAV"].(webdavfs.WebDAV)
		err = fs.Put(fileName, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case "S3":
		fs := h.App.FileSystems["S3"].(s3fs.S3)
		err = fs.Put(fileName, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	h.App.Session.Put(r.Context(), "flash", "File uploaded!")
	http.Redirect(w, r, "/files/upload?type="+uploadType, http.StatusSeeOther)
}

func getFileToUpload(r *http.Request, fieldName string) (string, error) {
	_ = r.ParseMultipartForm(10 << 20)
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	dst, err := os.Create(fmt.Sprintf("./tmp/%s", header.Filename))
	if err != nil {
		return "", err
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("./tmp/%s", header.Filename), nil
}

func (h *Handlers) DeleteFromFS(w http.ResponseWriter, r *http.Request) {
	var fs filesystems.FS
	fsType := r.URL.Query().Get("fs_type")
	item := r.URL.Query().Get("file")

	switch fsType {
	case "MINIO":
		f := h.App.FileSystems["MINIO"].(miniofs.Minio)
		fs = &f
	case "SFTP":
		f := h.App.FileSystems["SFTP"].(sftpfs.SFTP)
		fs = &f
	case "WEBDAV":
		f := h.App.FileSystems["WEBDAV"].(webdavfs.WebDAV)
		fs = &f
	case "S3":
		f := h.App.FileSystems["S3"].(s3fs.S3)
		fs = &f
	}

	deleted := fs.Delete([]string{item})
	if deleted {
		h.App.Session.Put(r.Context(), "flash", fmt.Sprintf("%s was deleted", item))
		http.Redirect(w, r, "/list-fs?fs-type="+fsType, http.StatusSeeOther)
	}
}

func (h *Handlers) Clicker(w http.ResponseWriter, r *http.Request) {
	err := h.render(w, r, "tester", nil, nil)
	if err != nil {
		h.App.ErrorLog.Println(err)
	}
}
