package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type Device struct {
	DevID          int
	UniqueID       string
	CurBattery     string
	CurTemp        string
	CurAttd        string
	CurPres        string
	RegisteredTime int
	LastSeen       int
	IP             string
}

type SensorData struct {
	ID         int
	DevID      int
	CurBattery string
	CurTemp    string
	CurAttd    string
	CurPres    string
	UpdateTime int
}

func main() {

	//initDB()
	db, err := sql.Open("mysql", "csgo:213q456qwe@tcp(sincos.icu:22205)/csgo")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if !tableExists(db, "LAB_devices") {
		// 创建表
		if err := createDeviceTable(db); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Table  created successfully.")
	}
	port := 3000
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/add_devices", addHandler)
	mux.HandleFunc("/update_data", updateData)
	fmt.Printf("Server listening on port %d...\n", port)
	err = http.ListenAndServe(":"+strconv.Itoa(port), mux)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
func createDeviceTable(db *sql.DB) error {

	queryDevice := `
		CREATE TABLE IF NOT EXISTS device (
			dev_id INT AUTO_INCREMENT PRIMARY KEY,
			unique_id VARCHAR(255),
			cur_battery VARCHAR(255),
			cur_temp VARCHAR(255),
			cur_attd VARCHAR(255),
			cur_pres VARCHAR(255),
			registeredTime TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			lastSeem TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			ip VARCHAR(255) NOT NULL
		);
	`
	_, err := db.Exec(queryDevice)
	if err != nil {
		return err
	}

	querySensorData := `
		CREATE TABLE IF NOT EXISTS sensor_data (
			id INT AUTO_INCREMENT PRIMARY KEY,
			dev_id INT,
			battery INT,
			temp INT,
			attd INT,
			pres INT,
			recordedTime TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (dev_id) REFERENCES device(dev_id)
		);
	`
	_, err = db.Exec(querySensorData)
	if err != nil {
		return err
	}

	return nil
}

func updateData(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data: "+err.Error(), http.StatusBadRequest)
		return
	}

	devId := r.Form.Get("dev_id")
	ComprehensiveData := r.Form.Get("SensorsJson")
	if devId == "" || ComprehensiveData == "" {
		// 返回 JSON 错误响应和自定义状态码
		errorResponse := map[string]interface{}{
			"msg":    "Please provide dev_id and SensorsJson!",
			"status": 400,
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}

	var sensorData SensorData
	err = json.Unmarshal([]byte(ComprehensiveData), &sensorData)
	if err != nil {
		errorResponse := map[string]interface{}{
			"msg":    "Failed to parse SensorsJson: " + err.Error(),
			"status": 402,
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}

	db, err := sql.Open("mysql", "csgo:213q456qwe@tcp(sincos.icu:22205)/csgo")
	if err != nil {
		// 返回 JSON 错误响应和自定义状态码
		errorResponse := map[string]interface{}{
			"error": "Failed to connect to database",
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonResponse)
		return
	}
	defer db.Close()

	// 检查设备是否存在
	if !ValueExists(db, "device", "dev_id", devId) {
		errorResponse := map[string]interface{}{
			"msg":    "Device not found",
			"status": 404,
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write(jsonResponse)
		return
	}

	err = UpdateDeviceData(db, devId, sensorData)
	if err != nil {
		errorResponse := map[string]interface{}{
			"msg":    "Failed to update device data: " + err.Error(),
			"status": 500,
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonResponse)
		return
	}
	err = UpdateSensorData(db, devId, sensorData)
	if err != nil {
		errorResponse := map[string]interface{}{
			"msg":    "Failed to update device data: " + err.Error(),
			"status": 500,
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonResponse)
		return
	}

	successResponse := map[string]interface{}{
		"status": "200",
		"msg":    err.Error(),
	}
	jsonResponse, _ := json.Marshal(successResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}
func UpdateSensorData(db *sql.DB, devId string, updateData SensorData) error {
	var setValues []string
	var params []interface{}

	if updateData.CurBattery != "" {
		setValues = append(setValues, "battery = ?")
		params = append(params, updateData.CurBattery)
	}
	if updateData.CurTemp != "" {
		setValues = append(setValues, "temp = ?")
		params = append(params, updateData.CurTemp)
	}
	if updateData.CurAttd != "" {
		setValues = append(setValues, "attd = ?")
		params = append(params, updateData.CurAttd)
	}
	if updateData.CurPres != "" {
		setValues = append(setValues, "pres = ?")
		params = append(params, updateData.CurPres)
	}

	if len(setValues) == 0 {
		return fmt.Errorf("no fields to update")
	}

	//build set
	setClause := "SET " + strings.Join(setValues, ", ")

	//build update
	query := "UPDATE sensor_data " + setClause + " WHERE dev_id = ?"
	params = append(params, devId)

	//exec update
	_, err := db.Exec(query, params...)
	if err != nil {
		return err
	}

	return nil
}
func UpdateDeviceData(db *sql.DB, devId string, updateData SensorData) error {
	var setValues []string
	var params []interface{}

	if updateData.CurBattery != "" {
		setValues = append(setValues, "cur_battery = ?")
		params = append(params, updateData.CurBattery)
	}
	if updateData.CurTemp != "" {
		setValues = append(setValues, "cur_temp = ?")
		params = append(params, updateData.CurTemp)
	}
	if updateData.CurAttd != "" {
		setValues = append(setValues, "cur_attd = ?")
		params = append(params, updateData.CurAttd)
	}
	if updateData.CurPres != "" {
		setValues = append(setValues, "cur_pres = ?")
		params = append(params, updateData.CurPres)
	}

	if len(setValues) == 0 {
		return fmt.Errorf("no fields to update")
	}

	//build set
	setClause := "SET " + strings.Join(setValues, ", ")

	//build update
	query := "UPDATE device " + setClause + " WHERE dev_id = ?"
	params = append(params, devId)

	//exec update
	_, err := db.Exec(query, params...)
	if err != nil {
		return err
	}

	return nil
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	// 解析表单数据
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// 检查是否传入了必需的字段 unique_id
	DeviceUniqueID := r.Form.Get("unique_id")
	if DeviceUniqueID == "" {
		// 返回 JSON 错误响应和自定义状态码
		errorResponse := map[string]interface{}{
			"msg":    "please post a DeviceUniqueID!",
			"status": 400,
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}

	// 连接到数据库
	db, err := sql.Open("mysql", "csgo:213q456qwe@tcp(sincos.icu:22205)/csgo")
	if err != nil {
		// 返回 JSON 错误响应和自定义状态码
		errorResponse := map[string]interface{}{
			"error": "Failed to connect to database",
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonResponse)
		return
	}
	defer db.Close()
	if ValueExists(db, "device", "unique_id", DeviceUniqueID) {
		errorResponse := map[string]interface{}{
			"status": 309,
			"msg":    "Device Exists.",
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-.Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonResponse)
		return

	}
	// 准备插入设备的 SQL 语句
	insertDeviceSQL := "INSERT INTO device (unique_id, ip) VALUES (?, ?)"
	result, err := db.Exec(insertDeviceSQL, DeviceUniqueID, r.RemoteAddr)
	if err != nil {
		// 返回 JSON 错误响应和自定义状态码
		errorResponse := map[string]interface{}{
			"status": 401,
			"msg":    "Failed to insert device into database",
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonResponse)
		return
	}
	lastInsertID, _ := result.LastInsertId()
	// 返回成功的 JSON 响应和状态码
	successResponse := map[string]interface{}{
		"status":    200,
		"unique_id": DeviceUniqueID,
		"dev_id":    lastInsertID,
	}
	jsonResponse, _ := json.Marshal(successResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Println("Error parsing form:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//token := r.Form.Get("token")
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	ipAddr := net.ParseIP(ip)

	ipType := "IPv4"
	if ipAddr != nil && ipAddr.To4() == nil {
		ipType = "IPv6"
	}

	response := map[string]interface{}{
		"ipType": ipType,
		"ip":     ip,
	}

	w.Header().Set("Content-Type", "application/json")
	/*
		if token == "" {
			response["message"] = "invalid token"
			response["status"] = 100
			w.WriteHeader(http.StatusForbidden)
		} else {
			response["rooms"] = rooms
			response["token"] = token
			response["status"] = 200
		}
	*/
	response["status"] = 200
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(jsonResponse)
}
func tableExists(db *sql.DB, tableName string) bool {
	var exists string
	query := fmt.Sprintf("SHOW TABLES LIKE '%s'", tableName)
	err := db.QueryRow(query).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		log.Fatal(err)
	}
	return exists == tableName
}
func ValueExists(db *sql.DB, tableName string, where string, value string) bool {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", tableName, where)
	var count int
	err := db.QueryRow(query, value).Scan(&count)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Fatal(err)
	}
	return count > 0
}
