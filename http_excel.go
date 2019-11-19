package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/360EntSecGroup-Skylar/excelize"
	"gopkg.in/mgo.v2"
)

// handleImportExcel 함수는 엑셀파일을 Import 하는 페이지 이다.
func handleImportExcel(w http.ResponseWriter, r *http.Request) {
	ssid, err := GetSessionID(r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	if ssid.AccessLevel == 0 {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	type recipe struct {
		User
		SessionID string
		Devmode   bool
	}
	rcp := recipe{}
	rcp.Devmode = *flagDevmode
	rcp.SessionID = ssid.ID
	session, err := mgo.Dial(*flagDBIP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer session.Close()
	rcp.User, err = getUser(session, ssid.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = TEMPLATES.ExecuteTemplate(w, "importexcel", rcp)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleImportExcel 함수는 엑셀파일을 Import 하는 페이지 이다.
func handlePresetExcel(w http.ResponseWriter, r *http.Request) {
	ssid, err := GetSessionID(r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	if ssid.AccessLevel == 0 {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	type recipe struct {
		User
		SessionID   string
		Devmode     bool
		Projectlist []string
		Files       []string
		SearchOption
	}
	rcp := recipe{}
	rcp.Devmode = *flagDevmode
	rcp.SessionID = ssid.ID
	tmp, err := userTemppath(ssid.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f, err := os.Open(tmp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	files, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, file := range files {
		rcp.Files = append(rcp.Files, file.Name())
	}
	session, err := mgo.Dial(*flagDBIP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer session.Close()
	rcp.SearchOption = handleRequestToSearchOption(r)
	rcp.User, err = getUser(session, ssid.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.Projectlist, err = OnProjectlist(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(rcp.Projectlist) == 0 {
		http.Redirect(w, r, "/noonproject", http.StatusSeeOther)
		return
	}
	err = TEMPLATES.ExecuteTemplate(w, "presetexcel", rcp)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleUploadExcel(w http.ResponseWriter, r *http.Request) {
	ssid, err := GetSessionID(r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	if ssid.AccessLevel == 0 {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}
	// dropzone setting
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Println(err)
	}
	defer file.Close()
	mimeType := header.Header.Get("Content-Type")
	switch mimeType {
	case "text/csv":
		data, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Fprintf(w, "%v", err)
			return
		}
		tmp, err := userTemppath(ssid.ID)
		if err != nil {
			fmt.Fprintf(w, "%v", err)
		}
		path := tmp + "/" + header.Filename // 업로드한 파일 리스트를 불러오기 위해 뒤에 붙는 Unixtime을 제거한다.
		err = ioutil.WriteFile(path, data, 0666)
		if err != nil {
			fmt.Fprintf(w, "%v", err)
			return
		}
		fmt.Println(path)
	case "application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": // MS-Excel, Google & Libre Excel
		data, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Fprintf(w, "%v", err)
			return
		}
		tmp, err := userTemppath(ssid.ID)
		if err != nil {
			fmt.Fprintf(w, "%v", err)
		}
		path := tmp + "/" + header.Filename // 업로드한 파일 리스트를 불러오기 위해 뒤에 붙는 Unixtime을 제거한다.
		err = ioutil.WriteFile(path, data, 0666)
		if err != nil {
			fmt.Fprintf(w, "%v", err)
			return
		}
	default:
		// 지원하지 않는 파일. 저장하지 않는다.
		log.Printf("Not support: %s", mimeType)
	}
}

// handlePresetExcelSubmit 함수는 excel 파일을 체크하고 분석 보고서로 Redirection 한다.
func handlePresetExcelSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Post Only", http.StatusMethodNotAllowed)
		return
	}
	ssid, err := GetSessionID(r)
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	if ssid.AccessLevel == 0 {
		http.Redirect(w, r, "/invalidaccess", http.StatusSeeOther)
		return
	}
	session, err := mgo.Dial(*flagDBIP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer session.Close()
	project := r.FormValue("project")
	filename := r.FormValue("filename")
	sheet := r.FormValue("sheet")
	overwrite := str2bool(r.FormValue("overwrite"))
	tmppath, err := userTemppath(ssid.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// .xlsx 파일을 읽는다.
	f, err := excelize.OpenFile(tmppath + "/" + filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var rows []Excelrow
	excelRows := f.GetRows(sheet)
	if len(excelRows) == 0 {
		http.Error(w, sheet+"값이 비어있습니다.", http.StatusBadRequest)
		return
	}
	for _, line := range excelRows {
		if len(line) != 15 {
			http.Error(w, "약속된 Cell 갯수가 다릅니다", http.StatusBadRequest)
			return
		}
		if line[0] == "샷네임" {
			continue
		}
		row := Excelrow{}
		row.Name = line[0]             // item name
		row.Shottype = line[1]         // shottype 2d,3d
		row.Note = line[2]             // 작업내용
		row.Comment = line[3]          // 수정사항
		row.Link = line[4]             // 링크자료(제목:경로)
		row.Ddline3D = line[5]         // 3D마감
		row.Ddline2D = line[6]         // 2D마감
		row.Findate = line[7]          // FIN날짜
		row.Finver = line[8]           // FIN버전
		row.Tags = line[9]             // 태그
		row.Rnum = line[10]            // 롤넘버
		row.HandleIn = line[11]        // 핸들IN
		row.HandleOut = line[12]       // 핸들OUT
		row.JustTimecodeIn = line[13]  // JUST타임코드IN
		row.JustTimecodeOut = line[14] // JUST타임코드OUT
		row.checkerror()               // 각 값을 에러체크한다.
		rows = append(rows, row)
	}

	type recipe struct {
		Project   string
		Filename  string
		Sheet     string
		Overwrite bool
		Rows      []Excelrow
		User
		SessionID string
		Devmode   bool
		SearchOption
	}
	rcp := recipe{}
	rcp.SessionID = ssid.ID
	rcp.Devmode = *flagDevmode

	rcp.SearchOption = handleRequestToSearchOption(r)
	rcp.User, err = getUser(session, ssid.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rcp.Rows = rows
	rcp.Project = project
	rcp.Filename = filename
	rcp.Sheet = sheet
	rcp.Overwrite = overwrite
	fmt.Println(rows)
	// project에 샷이 존재하는지 체크한다.
	// 각 col 값이 정상적인지 체크한다.
	// 결과 보고서를 만든다.
	err = TEMPLATES.ExecuteTemplate(w, "reportexcel", rcp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
