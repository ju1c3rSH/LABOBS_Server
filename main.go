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
	ID            int
	DevID         string
	Battery       int
	Temp          int
	Cttd          int
	Pres          int
	UpdateTime    string
	Methane       float32
	LPG           float32
	Smoke         float32
	Poisonous_Gas float32
}

type testData struct {
	CurBattery         float64
	CurTemp            float64
	CurAttd            float64
	CurPres            float64
	CurMethane         float64
	CurLPG             float64
	CurSmoke           float64
	CurPoisonousGasPPM float64
}

func main() {

	//initDB()
	db, err := sql.Open("mysql", "csgo:213q456qwe@tcp(sincos.icu:22205)/csgo")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if !tableExists(db, "LAB_devices") {
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
	mux.HandleFunc("/GetDeviceHistoryStatus", gdhsHandler)
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
func gdhsHandler(w http.ResponseWriter, r *http.Request) {
	// get device history sensors data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data: "+err.Error(), http.StatusBadRequest)
		return
	}
	devId := r.Form.Get("dev_id")
	exceptRows := r.Form.Get("exceptRows")
	if exceptRows == "" {
		exceptRows = "30"
	}

	if devId == "" {
		errorResponse := map[string]interface{}{
			"msg":    "Please provide dev_id!",
			"status": 400,
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}

	exceptRowsInt, err := strconv.Atoi(exceptRows)
	if err != nil {
		fmt.Println("Failed to convert exceptRows to integer:", err)
		return
	}
	fmt.Print(exceptRowsInt)

	db, err := sql.Open("mysql", ")")//这里是mysql服务器，需要补全
	if err != nil {
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
	query := fmt.Sprintf("SELECT * FROM sensor_data WHERE dev_id = ? ORDER BY id DESC LIMIT %d;\n", exceptRowsInt)
	rows, err := db.Query(query, devId)
	if err != nil {
		errorResponse := map[string]interface{}{
			"msg":    "Failed to get sensor data: " + err.Error(),
			"status": 405,
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}
	var sensorData []SensorData

	for rows.Next() {
		var id int
		var dev_id string
		var battery int
		var temp int
		var attd int
		var pres int
		var recordedTime string
		var methane float32
		var poisonous_gas_ppm float32
		var lpg float32
		var smoke float32
		if err := rows.Scan(&id, &dev_id, &battery, &temp, &attd, &pres, &recordedTime, &methane, &poisonous_gas_ppm, &lpg, &smoke); err != nil {
			fmt.Println("Error scanning row:", err)
			continue
		}
		var data SensorData
		data.ID = id
		data.Cttd = attd
		data.Battery = battery
		data.Temp = temp
		data.Pres = pres
		data.LPG = lpg
		data.Methane = methane
		data.Poisonous_Gas = poisonous_gas_ppm
		data.Smoke = smoke
		//data.UpdateTime = int(time.Now().UnixNano() / 1e6)
		data.UpdateTime = recordedTime

		sensorData = append(sensorData, data)

	}
	jsonResponse, err := json.Marshal(sensorData)
	if err != nil {
		errorResponse := map[string]interface{}{
			"msg":    "Failed to get sensor data: " + err.Error(),
			"status": 405,
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}
	successResponse := map[string]interface{}{
		"status": 200,
		"data":   sensorData,
	}
	jsonResponse, _ = json.Marshal(successResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
	//handle here

	defer rows.Close()

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
		fmt.Println("please post some data")
		errorResponse := map[string]interface{}{
			"msg":    "please post some data",
			"status": 400,
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonResponse)
		return
	}
	fmt.Println("ComprehensiveData: ", ComprehensiveData)
	fmt.Println("devId: ", devId)
	//fmt.Println(ComprehensiveData)
	//var sensorData SensorData
	var sensorData testData
	err = json.Unmarshal([]byte(ComprehensiveData), &sensorData)

	if err != nil {
		fmt.Println("Unmarshall: ", err.Error())
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

	//ComprehensiveData:  {'CurBattery': 100, 'CurTemp': 27.3944, 'CurPres': 100351.6, 'CurAttd': 77.12587}
	//devId:  zl3UOtVP
	//	invalid character '\'' looking for beginning of object key string
	//ComprehensiveData:  { "ID": 123, "DevID": "BE120de2", "CurBattery": 75, "CurTemp": 25, "CurAttd": 50, "CurPres": 1013, "UpdateTime": "2024-03-28T12:00:00Z" }
	//devId:  zl3UOtVP
	//	sd 25
	//	value ok
	//UpdateSensorData:  {123 BE120de2 75 25 50 1013 2024-03-28T12:00:00Z}

	fmt.Println("sd", sensorData.CurTemp)
	db, err := sql.Open("mysql", "csgo:213q456qwe@tcp(sincos.icu:22205)/csgo")
	if err != nil {
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

	if !ValueExists(db, "device", "unique_id", devId) {
		errorResponse := map[string]interface{}{
			"msg":    "Device not found",
			"status": 404,
		}
		fmt.Println("Device not found")
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write(jsonResponse)
		return
	} else {
		fmt.Println("value ok")
	}

	err = UpdateSensorData(db, devId, sensorData)
	if err != nil {
		errorResponse := map[string]interface{}{
			"msg":    "Failed to update sensor data: " + err.Error(),
			"status": 500,
		}
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonResponse)
		return
	}

	err = UpdateDeviceData(db, devId, sensorData)
	if err != nil {
		errorResponse := map[string]interface{}{
			"msg":    "Failed to update device data: " + err.Error(),
			"status": 500,
		}
		fmt.Println(err)
		jsonResponse, _ := json.Marshal(errorResponse)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonResponse)
		return
	}
	successResponse := map[string]interface{}{
		"status": "200",
		"msg":    "hi",
	}
	jsonResponse, _ := json.Marshal(successResponse)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

//	type SensorData struct {
//		ID         int
//		DevID      string
//		CurBattery int
//		CurTemp    int
//		CurAttd    int
//		CurPres    int
//		UpdateTime string
//	}
func UpdateSensorData(db *sql.DB, devId string, updateData testData) error {
	var setColumns []string
	var placeholders []string
	var params []interface{}
	fmt.Println("UpdateSensorData: ", updateData)
	if devId != "" {
		setColumns = append(setColumns, "dev_id")
		placeholders = append(placeholders, "?")
		params = append(params, devId)
	}
	if updateData.CurBattery != 0 {
		setColumns = append(setColumns, "battery")
		placeholders = append(placeholders, "?")
		params = append(params, updateData.CurBattery)
	}
	if updateData.CurLPG != 0 {
		setColumns = append(setColumns, "lpg")
		placeholders = append(placeholders, "?")
		params = append(params, updateData.CurLPG)
	}
	if updateData.CurMethane != 0 {
		setColumns = append(setColumns, "methane")
		placeholders = append(placeholders, "?")
		params = append(params, updateData.CurMethane)
	}
	if updateData.CurPoisonousGasPPM != 0 {
		setColumns = append(setColumns, "poisonous_gas_ppm")
		placeholders = append(placeholders, "?")
		params = append(params, updateData.CurPoisonousGasPPM)
	}
	if updateData.CurSmoke != 0 {
		setColumns = append(setColumns, "smoke")
		placeholders = append(placeholders, "?")
		params = append(params, updateData.CurSmoke)
	}
	if updateData.CurTemp != 0 {
		setColumns = append(setColumns, "temp")
		placeholders = append(placeholders, "?")
		params = append(params, updateData.CurTemp)
	}
	if updateData.CurAttd != 0 {
		setColumns = append(setColumns, "attd")
		placeholders = append(placeholders, "?")
		params = append(params, updateData.CurAttd)
	}
	if updateData.CurPres != 0 {
		setColumns = append(setColumns, "pres")
		placeholders = append(placeholders, "?")
		params = append(params, updateData.CurPres)
	}

	if len(setColumns) == 0 {
		return fmt.Errorf("no fields to update")
	}

	//build query
	columnClause := "(" + strings.Join(setColumns, ", ") + ")"
	placeholderClause := "VALUES (" + strings.Join(placeholders, ", ") + ")"
	query := "INSERT INTO sensor_data " + columnClause + " " + placeholderClause

	//exe query
	_, err := db.Exec(query, params...)
	if err != nil {
		return err
	}

	return nil
}
func UpdateDeviceData(db *sql.DB, devId string, updateData testData) error {
	var setColumns []string
	var params []interface{}

	// Check if device ID is provided
	if devId == "" {
		return fmt.Errorf("device ID is required")
	}

	// Check and add update data to setColumns and params
	if updateData.CurBattery != 0 {
		setColumns = append(setColumns, "cur_battery = ?")
		params = append(params, updateData.CurBattery)
	}
	if updateData.CurLPG != 0 {
		setColumns = append(setColumns, "cur_lpg = ?")
		params = append(params, updateData.CurLPG)
	}
	if updateData.CurMethane != 0 {
		setColumns = append(setColumns, "cur_methane = ?")
		params = append(params, updateData.CurMethane)
	}
	if updateData.CurPoisonousGasPPM != 0 {
		setColumns = append(setColumns, "cur_poisonous_gas_ppm = ?")
		params = append(params, updateData.CurPoisonousGasPPM)
	}
	if updateData.CurSmoke != 0 {
		setColumns = append(setColumns, "cur_smoke = ?")
		params = append(params, updateData.CurSmoke)
	}
	if updateData.CurTemp != 0 {
		setColumns = append(setColumns, "cur_temp = ?")
		params = append(params, updateData.CurTemp)
	}
	if updateData.CurAttd != 0 {
		setColumns = append(setColumns, "cur_attd = ?")
		params = append(params, updateData.CurAttd)
	}
	if updateData.CurPres != 0 {
		setColumns = append(setColumns, "cur_pres = ?")
		params = append(params, updateData.CurPres)
	}

	// Check if there are fields to update
	if len(setColumns) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Construct the SQL query
	setClause := strings.Join(setColumns, ", ")
	query := "UPDATE device SET " + setClause + " WHERE unique_id = ?"

	// Add device ID to params
	params = append(params, devId)

	// Execute the update query
	_, err := db.Exec(query, params...)
	if err != nil {
		return err
	}

	return nil
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	DeviceUniqueID := r.Form.Get("unique_id")
	if DeviceUniqueID == "" {
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

	db, err := sql.Open("mysql", "csgo:213q456qwe@tcp(sincos.icu:22205)/csgo")
	if err != nil {
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

	insertDeviceSQL := "INSERT INTO device (unique_id, ip) VALUES (?, ?)"
	result, err := db.Exec(insertDeviceSQL, DeviceUniqueID, r.RemoteAddr)
	if err != nil {
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
