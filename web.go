package main

import (
	"fmt"
	"net/http"
	"os"
	"io/ioutil"
	"github.com/julienschmidt/httprouter"
	"log"
	"database/sql"
	_ "github.com/lib/pq"
	"time"
	_ "text/scanner"
	"math"
	_ "syscall"
	"encoding/json"
	"strconv"
	"strings"
)

const (
	DB_USER     = "adminfjxusjc"
	DB_PASSWORD = "JIw6nIgd-Mck"
	DB_NAME     = "golang"
)
type Gift struct {
	EventName string
	BonusName string
	GemsCount int
	MoneyCount int
	TramLivesCount int
	Multiplier float32
	MultiplyByMoney bool
	TramSkinName string
	ConductorSkinName string
	FibonacciCoef int
	IsSumm bool
	CanSendManyTimes bool
}

type Event struct {
	EventType string
	EventNumberValue int
	EventStringValue string
}

type Bonus struct {
	BonusType string
	Count int
}

type Constants struct {
	TramLivesRestorePeriodInMinutes int64
	GemsStartCount int
	MoneyStartCount int
}

type TramSkin struct {
	SkinName string
	HP int
}

type Combination struct {
	Name string
	Reward int
}

var DatabaseConnection *sql.DB
var Random *os.File
var configFile []byte
var giftsArr []Gift
var tramSkins []TramSkin
var currentConstants *Constants
var combination1 Combination
var combination2 Combination
var combination3 Combination
var combination2and2 Combination
var combination3and3 Combination
var combination2and3 Combination
var combination2and2and2 Combination
var combination4 Combination
var combination4and2 Combination
var combination5 Combination
var combination6 Combination
var combinationsInitialized bool

func main() {
	_,present := os.LookupEnv("OPENSHIFT_GO_IP")
	if !present {
		os.Setenv("OPENSHIFT_GO_IP", "127.0.0.1")
		os.Setenv("OPENSHIFT_GO_PORT", "8080")
		os.Setenv("OPENSHIFT_POSTGRESQL_DB_HOST", "127.0.0.1")
		os.Setenv("OPENSHIFT_POSTGRESQL_DB_PORT", "5432")
	}
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		DB_USER, DB_PASSWORD, DB_NAME, os.Getenv("OPENSHIFT_POSTGRESQL_DB_HOST"), os.Getenv("OPENSHIFT_POSTGRESQL_DB_PORT"))
	var err error
	DatabaseConnection, err = sql.Open("postgres", dbinfo)
	handleError(err, nil)
	DatabaseConnection.SetMaxIdleConns(16)
	DatabaseConnection.SetMaxOpenConns(16)
	createDatabase()
	router := httprouter.New()
	router.GET("/", index)
	router.GET("/config/:action", configManage)
	router.POST("/user/:action", userManage)
	router.POST("/event/:action", eventUnlock)
	router.GET("/event/:userid", eventsList)
	router.GET("/bonus/:userid", bonusesList)
	router.POST("/bonus/use/:bonus", decreaseBonus)
	router.GET("/resources/:userid", resourcesList)
	router.POST("/tramlives/decrease", decreaseTramLives)
	router.GET("/tramlives/get/:userid", checkTramLives)
	router.POST("/combination/:action", combinationManage)
	bind := fmt.Sprintf("%s:%s", os.Getenv("OPENSHIFT_GO_IP"), os.Getenv("OPENSHIFT_GO_PORT"))
	fmt.Printf("listening on %s...", bind)
	log.Fatal(http.ListenAndServe(bind, router))
}

func createDatabase() {
	_, err := DatabaseConnection.Query("CREATE TABLE IF NOT EXISTS Users (userid TEXT PRIMARY KEY, currentToken TEXT, gemsCount INTEGER, moneyCount INTEGER, previousCombinationCount INTEGER, previousCombinationName TEXT)")
	handleError(err, nil)
	_, err = DatabaseConnection.Query("CREATE TABLE IF NOT EXISTS Bonuses (id SERIAL PRIMARY KEY, bonusType TEXT, userid TEXT REFERENCES Users(userid) ON DELETE CASCADE ON UPDATE CASCADE, bonusCount INTEGER)")
	handleError(err, nil)
	_, err = DatabaseConnection.Query("CREATE TABLE IF NOT EXISTS ConductorSkins (id SERIAL PRIMARY KEY, conductorType TEXT, userid TEXT REFERENCES Users(userid) ON DELETE CASCADE ON UPDATE CASCADE, isCurrent BOOLEAN)")
	handleError(err, nil)
	_, err = DatabaseConnection.Query("CREATE TABLE IF NOT EXISTS TramSkins (id SERIAL PRIMARY KEY, tramType TEXT, userid TEXT REFERENCES Users(userid) ON DELETE CASCADE ON UPDATE CASCADE, isCurrent BOOLEAN)")
	handleError(err, nil)
	_, err = DatabaseConnection.Query("CREATE TABLE IF NOT EXISTS Events (id SERIAL PRIMARY KEY, eventType TEXT, userid TEXT REFERENCES Users(userid) ON DELETE CASCADE ON UPDATE CASCADE, eventNumberValue INTEGER, eventStringValue TEXT, prevFibValue INTEGER)")
	handleError(err, nil)
	_, err = DatabaseConnection.Query("CREATE TABLE IF NOT EXISTS TramLives (userid TEXT REFERENCES Users(userid) ON DELETE CASCADE ON UPDATE CASCADE PRIMARY KEY, livesCount INTEGER, updateTimestamp TIMESTAMPTZ)")
	handleError(err, nil)
}

func handleError(err error, w http.ResponseWriter) {
	if err != nil {
		if(w != nil) {
			response := fmt.Sprintf("{\"error\": \""+err.Error()+"\"}")
			writeJsonResponse(response, w)
		}
		//panic(err)
	}
}

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}



func writeJsonResponse(jsonString string, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, X-Access-Token, X-Application-Name, X-Request-Sent-Time")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprintf(w, jsonString)
}

func createToken() string {
	return fmt.Sprintf("%s", createUUID())
}

func configManage(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	action := params.ByName("action")
	if action == "get" {
		var err error
		if configFile == nil {
			configFile, err = ioutil.ReadFile("config.json")
		}
		handleError(err, w)
		writeJsonResponse(fmt.Sprintf(string(configFile)), w)
		return
	}
	if action == "version" {
		var err error
		if configFile == nil {
			configFile, err = ioutil.ReadFile("config.json")
		}
		handleError(err, w)
		decodedConfig := make(map[string]interface{})
		err = json.Unmarshal(configFile, &decodedConfig)
		handleError(err, w)
		version := decodedConfig["version"]
		response := fmt.Sprintf("{\"configVersion\": \"%s\"}", version)
		writeJsonResponse(response, w)
		return
	}
	fmt.Fprintf(w, "Incorrect action\n")
}

func loadTramSkins() {
	if tramSkins == nil {
		skinsConfigFile, err := ioutil.ReadFile("tramskins_config.json")
		handleError(err, nil)
		err = json.Unmarshal(skinsConfigFile, &tramSkins)
		handleError(err, nil)
	}
}

func combinationManage(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	userid := userIdFromPostRequest(w, r)
	if userid == "" {
		return
	}
	if !combinationsInitialized {
		config, err := ioutil.ReadFile("combinations_config.json")
		handleError(err, w)
		var combinations []Combination
		err = json.Unmarshal(config, &combinations)
		for index,combination := range combinations {
			switch index {
			case 0 :
				combination1 = combination
			case 1 :
				combination2 = combination
			case 2 :
				combination3 = combination
			case 3 :
				combination2and2 = combination
			case 4 :
				combination3and3 = combination
			case 5 :
				combination2and3 = combination
			case 6 :
				combination2and2and2 = combination
			case 7 :
				combination4 = combination
			case 8 :
				combination4and2 = combination
			case 9 :
				combination5 = combination
			case 10 :
				combination6 = combination
			}
		}
		handleError(err, w)
		combinationsInitialized = true
	}
	action := params.ByName("action")
	if action == "start" {
		_, err := DatabaseConnection.Query("UPDATE Users SET previousCombinationCount=0 WHERE userid=$1", userid)
		handleError(err, w)
		_, err = DatabaseConnection.Query("UPDATE Users SET previousCombinationName='' WHERE userid=$1", userid)
		handleError(err, w)
		if err == nil {
			response := fmt.Sprintf("{\"result\": \"true\"}")
			writeJsonResponse(response, w)
		}
		return
	}
	if action == "send" {
		passengersArray := r.PostFormValue("passengersArray")
		parsedPassengersArray := strings.Split(passengersArray, ",")
		foundCount := make(map[string]int)
		for _, passenger := range parsedPassengersArray  {
			if _,exists := foundCount[passenger]; exists {
				foundCount[passenger]++
			} else {
				foundCount[passenger] = 1
			}
		}
		twoFound := false
		secondTwoFound := false
		thirdTwoFound := false
		threeFound := false
		secondThreeFound := false
		fourFound := false
		fiveFound := false
		sixFound := false
		for _,k := range foundCount {
			switch k {
			case 2:
				if secondTwoFound {
					thirdTwoFound = true
				} else {
					if twoFound {
						secondTwoFound = true
					} else {
						twoFound = true
					}
				}
			case 3:
				if threeFound {
					secondThreeFound = true
				} else {
					threeFound = true
				}
			case 4:
				fourFound = true
			case 5:
				fiveFound = true
			case 6:
				sixFound = true
			}
		}
		if sixFound {
			addCombinationReward(w, combination6, userid)
			return
		}
		if fiveFound {
			addCombinationReward(w, combination5, userid)
			return
		}
		if fourFound && twoFound {
			addCombinationReward(w, combination4and2, userid)
			return
		}
		if fourFound {
			addCombinationReward(w, combination4, userid)
			return
		}
		if secondThreeFound {
			addCombinationReward(w, combination3and3, userid)
			return
		}
		if threeFound && twoFound {
			addCombinationReward(w, combination2and3, userid)
			return
		}
		if threeFound {
			addCombinationReward(w, combination3, userid)
			return
		}
		if thirdTwoFound {
			addCombinationReward(w, combination2and2and2, userid)
			return
		}
		if secondTwoFound {
			addCombinationReward(w, combination2and2, userid)
			return
		}
		if twoFound {
			addCombinationReward(w, combination2, userid)
			return
		}
		addSingleCombinationReward(w, len(foundCount), userid)
		return
	}
}

func addCombinationReward(w http.ResponseWriter, combination Combination, userid string) {
	var name string
	var count int
	err := DatabaseConnection.QueryRow("SELECT previousCombinationName, previousCombinationCount FROM Users WHERE userid=$1", userid).Scan(&name, &count)
	handleError(err, w)
	if name == combination.Name {
		count = count + 1
	} else {
		count = 1
		name = combination.Name
	}
	_, err = DatabaseConnection.Query("UPDATE Users SET previousCombinationCount=$1 WHERE userid=$2", count, userid)
	handleError(err, w)
	_, err = DatabaseConnection.Query("UPDATE Users SET previousCombinationName=$1 WHERE userid=$2", name, userid)
	handleError(err, w)
	sum := strconv.Itoa(combination.Reward*count)
	response := fmt.Sprintf("{\"result\": \"true\", \"reward\": "+sum+", \"name\": \""+combination.Name+"\"}")
	writeJsonResponse(response, w)
}

func addSingleCombinationReward(w http.ResponseWriter, passengersCount int, userid string) {
	count := 0
	name := combination1.Name
	_, err := DatabaseConnection.Query("UPDATE Users SET previousCombinationCount=$1 WHERE userid=$2", count, userid)
	handleError(err, w)
	_, err = DatabaseConnection.Query("UPDATE Users SET previousCombinationName=$1 WHERE userid=$2", name, userid)
	handleError(err, w)
	sum := strconv.Itoa(combination1.Reward*passengersCount)
	response := fmt.Sprintf("{\"result\": \"true\", \"reward\": "+sum+", \"name\": \""+combination1.Name+"\"}")
	writeJsonResponse(response, w)
}

func eventUnlock(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	userid := userIdFromPostRequest(w, r)
	if userid == "" {
		return
	}
	eventName := r.PostFormValue("eventName")
	if eventName == "" {
		response := fmt.Sprintf("{\"error\": \"incorrectEventName\"}")
		writeJsonResponse(response, w)
		return
	}

	stringParameter := r.PostFormValue("stringParameter")
	intParameterRepresentation := r.PostFormValue("intParameter")
	intParameter, err := strconv.Atoi(intParameterRepresentation)
	handleError(err, w)
	if giftsArr == nil {
		eventConfigFile, err := ioutil.ReadFile("event_config.json")
		handleError(err, w)
		err = json.Unmarshal(eventConfigFile, &giftsArr)
		handleError(err, w)
	}
	var currentNumberParameter int
	var currentFibParameter int
	numberError := DatabaseConnection.QueryRow("SELECT eventnumbervalue, prevFibValue FROM Events WHERE userid=$1 AND eventType=$2 AND eventstringvalue=$3", userid, eventName, stringParameter).Scan(&currentNumberParameter, &currentFibParameter)
	for _,currentGift := range giftsArr {
		if currentGift.EventName == eventName {
			giftJson := ""
			previousFibonacciNumber := 0
			if intParameter == currentNumberParameter && !currentGift.IsSumm && !currentGift.CanSendManyTimes {
				response := fmt.Sprintf("{\"error\": \"this event already sent\"}")
				writeJsonResponse(response, w)
				return
			}
			if currentGift.FibonacciCoef == 0 {
				if intParameter > currentNumberParameter || currentGift.CanSendManyTimes {
					giftJson = currentGift.addGift(w, userid, intParameter)
				}
			} else {
				if intParameter > currentNumberParameter || currentGift.IsSumm {
					var value2compare int
					if currentGift.IsSumm {
						value2compare = currentNumberParameter + intParameter
					} else {
						value2compare = intParameter
					}

					currentFibonacciNumber := currentGift.FibonacciCoef
					minimalFibNumFound := false
					for minimalFibNumFound == false {
						if(currentFibonacciNumber > value2compare) {
							minimalFibNumFound = true
						} else {
							buf := previousFibonacciNumber
							previousFibonacciNumber = currentFibonacciNumber
							currentFibonacciNumber = currentFibonacciNumber + buf
						}
					}
					if value2compare >= previousFibonacciNumber && previousFibonacciNumber != currentFibParameter {
						var valueForCalculateGift int
						if currentGift.IsSumm {
							valueForCalculateGift = value2compare - previousFibonacciNumber
						} else {
							valueForCalculateGift = intParameter
						}
						giftJson = currentGift.addGift(w, userid, valueForCalculateGift)
					}
				}
			}
			if numberError == sql.ErrNoRows {
				_, err = DatabaseConnection.Query("INSERT INTO Events(eventType, userid, eventNumberValue, eventStringValue, prevFibValue) VALUES ($1, $2, $3, $4, 0)", eventName, userid, intParameter, stringParameter)
				handleError(err, w)
			} else {
				if intParameter > currentNumberParameter || currentGift.IsSumm {
					if currentGift.IsSumm {
						intParameter += currentNumberParameter
					}
					_, err = DatabaseConnection.Query("UPDATE Events SET eventNumberValue=$1 WHERE userid=$2 AND eventType=$3 AND eventstringvalue=$4", intParameter, userid, eventName, stringParameter)
					handleError(err, w)
					_, err = DatabaseConnection.Query("UPDATE Events SET prevFibValue=$1 WHERE userid=$2 AND eventType=$3 AND eventstringvalue=$4", previousFibonacciNumber, userid, eventName, stringParameter)
					handleError(err, w)
				}
			}

			var response string
			if(len(giftJson) > 0) {
				response = fmt.Sprintf("{\"result\": \"true\"," + giftJson + "}")
			} else {
				response = fmt.Sprintf("{\"result\": \"true\"}")
			}

			writeJsonResponse(response, w)
			return
		}
	}
	response := fmt.Sprintf("{\"error\": \"unknown event name\"}")
	writeJsonResponse(response, w)
}

func decreaseTramLives(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	userid := userIdFromPostRequest(w, r)
	if userid == "" {
		return
	}
	var currentTramLivesCount int
	err := DatabaseConnection.QueryRow("SELECT livesCount FROM TramLives WHERE userid=$1", userid).Scan(&currentTramLivesCount)
	handleError(err, w)
	if currentTramLivesCount > 0 {
		currentTramLivesCount--
		_, err = DatabaseConnection.Query("UPDATE TramLives SET livesCount=$1 WHERE userid=$2", currentTramLivesCount, userid)
		handleError(err, w)
	}
	response := fmt.Sprintf("{\"tramLivesCount\": %d }", currentTramLivesCount)
	writeJsonResponse(response, w)
}

func decreaseBonus(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	userid := userIdFromPostRequest(w, r)
	if userid == "" {
		return
	}
	bonusType := params.ByName("bonus")
	var currentBonusCount int
	err := DatabaseConnection.QueryRow("SELECT bonusCount FROM Bonuses WHERE userid=$1 AND bonusType=$2", userid, bonusType).Scan(&currentBonusCount)
	if err == sql.ErrNoRows {
		response := fmt.Sprintf("{\"error\" : \"bonus not found\"}")
		writeJsonResponse(response, w)
		return
	} else {
		handleError(err, w)
	}
	currentBonusCount -= 1
	if currentBonusCount > 1 {
		_, err = DatabaseConnection.Query("UPDATE Bonuses SET bonusCount=$1 WHERE userid=$2 AND bonusType=$3", currentBonusCount, userid, bonusType)
		handleError(err, w)
	} else {
		_, err = DatabaseConnection.Query("DELETE FROM Bonuses WHERE userid=$1 AND bonusType=$2", userid, bonusType)
		handleError(err, w)
	}
	response := fmt.Sprintf("{\"result\": \"true\"}")
	writeJsonResponse(response, w)
}


func checkTramLives(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	userid := useridFromGetRequest(w, params)
	if userid == "" {
		return
	}
	var currentUpdateTimestamp *time.Time
	var livesCount int
	err := DatabaseConnection.QueryRow("SELECT updateTimeStamp, livesCount FROM TramLives WHERE userid=$1", userid).Scan(&currentUpdateTimestamp, &livesCount)
	handleError(err, w)
	interval := time.Now().Sub(*currentUpdateTimestamp)
	durationInMinutes := int64(interval/time.Minute)
	createConstantsIfNeeded()
	restoredLives := int(durationInMinutes/currentConstants.TramLivesRestorePeriodInMinutes)
	if restoredLives == 0 {
		response := fmt.Sprintf("{\"tramLivesCount\": %d }", livesCount)
		writeJsonResponse(response, w)
		return
	}
	restoredLives += livesCount
	var currentSkinName string
	loadTramSkins()
	err = DatabaseConnection.QueryRow("SELECT tramType FROM TramSkins WHERE userid=$1 AND isCurrent=TRUE", userid).Scan(&currentSkinName)
	handleError(err, w)
	maxHP := 0
	for _, currentSkinConfig := range tramSkins {
		if currentSkinConfig.SkinName == currentSkinName {
			maxHP = currentSkinConfig.HP
			break
		}
	}
	if restoredLives > maxHP {
		restoredLives = maxHP
	}
	_, err = DatabaseConnection.Query("UPDATE TramLives SET livesCount=$1 WHERE userid=$2", restoredLives, userid)
	handleError(err, w)
	_, err = DatabaseConnection.Query("UPDATE TramLives SET updateTimeStamp=$1 WHERE userid=$2", time.Now().Format(time.RFC3339), userid)
	handleError(err, w)
	response := fmt.Sprintf("{\"tramLivesCount\": %d }", restoredLives)
	writeJsonResponse(response, w)
}

func userIdFromPostRequest(w http.ResponseWriter, r *http.Request) string {
	token := r.PostFormValue("token")
	if token == "" {
		response := fmt.Sprintf("{\"error\": \"incorrectToken\"}",)
		writeJsonResponse(response, w)
		return ""
	}
	var userid string
	err := DatabaseConnection.QueryRow("SELECT userid FROM Users WHERE currentToken=$1", token).Scan(&userid)
	if err == sql.ErrNoRows {
		response := fmt.Sprintf("{\"error\": \"incorrectToken\"}",)
		writeJsonResponse(response, w)
		return ""
	}
	handleError(err, w)
	return userid
}

func (gift2add Gift) addGift(w http.ResponseWriter, userid string, baseCount int) string {
	result := ""
	totalGemsCount := 0
	totalMoneyCount := 0
	if len(gift2add.BonusName) > 0 {
		var currentBonusCount int
		err := DatabaseConnection.QueryRow("SELECT bonusCount FROM Bonuses WHERE userid=$1 AND bonusType=$2", userid, gift2add.BonusName).Scan(&currentBonusCount)
		if err == sql.ErrNoRows {
			_, err = DatabaseConnection.Query("INSERT INTO Bonuses(bonusType, userid, bonusCount) VALUES ($1, $2, 1)", gift2add.BonusName, userid)
			handleError(err, w)
		} else {
			handleError(err, w)
		}
		currentBonusCount += 1
		_, err = DatabaseConnection.Query("UPDATE Bonuses SET bonusCount=$1 WHERE userid=$2 AND bonusType=$3", currentBonusCount, userid, gift2add.BonusName)
		handleError(err, w)
		result = " \"bonus\": \"" + gift2add.BonusName + "\""
	}

	if gift2add.GemsCount > 0 {
		currentGemsCount := getCurrentGemsCount(w, userid)
		currentGemsCount += gift2add.GemsCount
		totalGemsCount = gift2add.GemsCount
		_, err := DatabaseConnection.Query("UPDATE Users SET gemsCount=$1 WHERE userid=$2", currentGemsCount, userid)
		handleError(err, w)
	}
	if gift2add.MoneyCount > 0 {
		var currentMoneyCount int
		err := DatabaseConnection.QueryRow("SELECT moneyCount FROM Users WHERE userid=$1", userid).Scan(&currentMoneyCount)
		handleError(err, w)
		currentMoneyCount += gift2add.MoneyCount
		totalMoneyCount = gift2add.MoneyCount
		_, err = DatabaseConnection.Query("UPDATE Users SET moneyCount=$1 WHERE userid=$2", currentMoneyCount, userid)
		handleError(err, w)
	}
	if gift2add.Multiplier > 0 {
		if gift2add.MultiplyByMoney {
			updatedValue := int(gift2add.Multiplier * float32(baseCount))
			if updatedValue < 1 {
				updatedValue = 1
			}
			currentCount := getCurrentMoneyCount(w, userid)
			totalMoneyCount += updatedValue
			updatedValue += currentCount
			_, err := DatabaseConnection.Query("UPDATE Users SET moneyCount=$1 WHERE userid=$2", updatedValue, userid)
			handleError(err, w)
		} else {
			updatedValue := int(gift2add.Multiplier * float32(baseCount))
			if updatedValue < 1 {
				updatedValue = 1
			}
			currentGemsCount := getCurrentGemsCount(w, userid)
			totalGemsCount += updatedValue
			updatedValue += currentGemsCount
			_, err := DatabaseConnection.Query("UPDATE Users SET gemsCount=$1 WHERE userid=$2", updatedValue, userid)
			handleError(err, w)
		}
	}
	if totalGemsCount > 0 {
		result = updateRecordJson(result, "gems", totalGemsCount)
	}
	if totalMoneyCount > 0 {
		result = updateRecordJson(result, "money", totalMoneyCount)
	}
	if(gift2add.TramLivesCount > 0) {
		var currentTramLivesCount int
		err := DatabaseConnection.QueryRow("SELECT livesCount FROM TramLives WHERE userid=$1", userid).Scan(&currentTramLivesCount)
		handleError(err, w)
		currentTramLivesCount += gift2add.TramLivesCount
		result = updateRecordJson(result, "tramlives", gift2add.TramLivesCount)
		_, err = DatabaseConnection.Query("UPDATE TramLives SET livesCount=$1 WHERE userid=$2", currentTramLivesCount, userid)
		handleError(err, w)
	}
	if(len(gift2add.TramSkinName) > 0) {
		tramRows, err := DatabaseConnection.Query("SELECT tramType, isCurrent FROM TramSkins WHERE userid=$1", userid)
		if err != sql.ErrNoRows {
			handleError(err, w)
		}
		skinAlreadyExists := false
		for tramRows.Next() {
			var tramName string
			var isCurrent bool
			err := tramRows.Scan(&tramName, &isCurrent)
			handleError(err, w)
			if tramName == gift2add.TramSkinName {
				skinAlreadyExists = true
				break
			}
		}
		if !skinAlreadyExists {
			_, err = DatabaseConnection.Query("INSERT INTO TramSkins(TramType, userid, isCurrent) VALUES ($1, $2, TRUE)", gift2add.TramSkinName, userid)
			handleError(err, w)
			for tramRows.Next() {
				var tramName string
				var isCurrent bool
				err := tramRows.Scan(&tramName, &isCurrent)
				handleError(err, w)
				if isCurrent && tramName != gift2add.TramSkinName {
					_, err = DatabaseConnection.Query("UPDATE TramSkins SET isCurrent=FALSE WHERE userid=$1 AND tramType=$2", userid, tramName)
					handleError(err, w)
				}
			}
			if(len(result) > 0) {
				result = result + ", \"tramSkin\": \"" + gift2add.TramSkinName + "\""
			} else {
				result = "\"tramSkin\": \"" + gift2add.TramSkinName + "\""
			}
		}
		tramRows.Close()
	}
	if(len(gift2add.ConductorSkinName) > 0) {
		conductorRows, err := DatabaseConnection.Query("SELECT conductorType, isCurrent FROM ConductorSkins WHERE userid=$1", userid)
		if err != sql.ErrNoRows {
			handleError(err, w)
		}
		skinAlreadyExists := false
		for conductorRows.Next() {
			var conductorName string
			var isCurrent bool
			err := conductorRows.Scan(&conductorName, &isCurrent)
			handleError(err, w)
			if conductorName == gift2add.ConductorSkinName {
				skinAlreadyExists = true
				break
			}
		}
		if !skinAlreadyExists {
			_, err = DatabaseConnection.Query("INSERT INTO ConductorSkins(ConductorType, userid, isCurrent) VALUES ($1, $2, TRUE)", gift2add.ConductorSkinName, userid)
			handleError(err, w)
			for conductorRows.Next() {
				var conductorName string
				var isCurrent bool
				err := conductorRows.Scan(&conductorName, &isCurrent)
				handleError(err, w)
				if isCurrent && conductorName != gift2add.ConductorSkinName {
					_, err = DatabaseConnection.Query("UPDATE ConductorSkins SET isCurrent=FALSE WHERE userid=$1 AND conductorType=$2", userid, conductorName)
					handleError(err, w)
				}
			}
			if(len(result) > 0) {
				result = result + ", \"conductorSkin\": \"" + gift2add.ConductorSkinName + "\""
			} else {
				result = "\"conductorSkin\": \"" + gift2add.ConductorSkinName + "\""
			}
		}
		conductorRows.Close()
	}
	return result
}

func updateRecordJson(currentJson string, fieldName string, fieldValue int) string {
	if len(currentJson) > 0 {
		currentJson = currentJson + ",\"" + fieldName + "\": " + strconv.Itoa(fieldValue)
	} else {
		currentJson = "\"" + fieldName + "\": " + strconv.Itoa(fieldValue)
	}
	return 	currentJson
}

func getCurrentGemsCount(w http.ResponseWriter, userid string) int {
	var currentGemsCount int
	err := DatabaseConnection.QueryRow("SELECT gemsCount FROM Users WHERE userid=$1", userid).Scan(&currentGemsCount)
	handleError(err, w)
	return currentGemsCount
}

func getCurrentMoneyCount(w http.ResponseWriter, userid string) int {
	var currentCount int
	err := DatabaseConnection.QueryRow("SELECT moneyCount FROM Users WHERE userid=$1", userid).Scan(&currentCount)
	handleError(err, w)
	return currentCount
}

func createConstantsIfNeeded() {
	if currentConstants == nil {
		constConfigFile, err := ioutil.ReadFile("constants.json")
		handleError(err, nil)
		err = json.Unmarshal(constConfigFile, &currentConstants)
		handleError(err, nil)
	}
}

func userManage(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	action := params.ByName("action")
	uuidFromRequest := r.PostFormValue("uuid")
	if uuidFromRequest == "" {
		response := fmt.Sprintf("{\"error\":\"incorrect uuid\"}")
		writeJsonResponse(response, w)
		return
	}
	if action == "register" {
		var result string
		err := DatabaseConnection.QueryRow("SELECT userid FROM Users WHERE userid=$1", uuidFromRequest).Scan(&result)
		if err == sql.ErrNoRows {
			createConstantsIfNeeded()
			tokenUUID := createToken()
			_, err = DatabaseConnection.Query("INSERT INTO Users VALUES ($1, $2, $3, $4)", uuidFromRequest, tokenUUID, currentConstants.GemsStartCount, currentConstants.MoneyStartCount)
			handleError(err, w)
			loadTramSkins()
			var tramLivesStartCount int
			for _,currentSkin := range tramSkins {
				if currentSkin.SkinName == "standard" {
					tramLivesStartCount = currentSkin.HP
					break
				}
			}
			_, err = DatabaseConnection.Query("INSERT INTO TramLives VALUES ($1, $2, $3)", uuidFromRequest, tramLivesStartCount, time.Now().Format(time.RFC3339))
			handleError(err, w)
			_, err = DatabaseConnection.Query("INSERT INTO TramSkins(TramType, userid, isCurrent) VALUES ('standard', $1, TRUE)", uuidFromRequest)
			handleError(err, w)
			response := fmt.Sprintf("{\"token\": \"%s\"}", tokenUUID)
			writeJsonResponse(response, w)
			return
		}
		if err == nil {
			freeUUID := createUUID()
			identicalFound := true
			for identicalFound {
				checkNewIdErr := DatabaseConnection.QueryRow("SELECT userid FROM Users WHERE userid=$1", freeUUID).Scan(&result)
				if checkNewIdErr == sql.ErrNoRows {
					identicalFound = false
				}
			}
			response := fmt.Sprintf("{\"freeuuid\": \"%s\"}", freeUUID)
			writeJsonResponse(response, w)
			return
		}
		return
	}
	if action == "authorize" {
		oldToken := r.PostFormValue("token")
		if oldToken == "" {
			response := fmt.Sprintf("{\"result\": \"false\", \"error\": \"incorrectToken\"}",)
			writeJsonResponse(response, w)
			return
		}
		var userid string
		err := DatabaseConnection.QueryRow("SELECT userid FROM Users WHERE currentToken=$1 AND userid=$2", oldToken, uuidFromRequest).Scan(&userid)
		if err == nil {
			tokenUUID := createToken()
			_,err = DatabaseConnection.Exec(fmt.Sprintf("UPDATE Users SET currentToken = '%s' where userid='%s'", tokenUUID, userid))
			handleError(err, w)
			response := fmt.Sprintf("{\"token\": \"%s\"}", tokenUUID)
			writeJsonResponse(response, w)
			return
		}
		if err == sql.ErrNoRows {
			response := fmt.Sprintf("{\"error\": \"incorrectToken\"}",)
			writeJsonResponse(response, w)
			return
		} else {
			handleError(err, w)
		}
		return
	}
	if action == "bind" {
		bindid := r.PostFormValue("bindid")
		if bindid == "" {
			response := fmt.Sprintf("{\"error\": \"incorrect bind id\"}",)
			writeJsonResponse(response, w)
			return
		}
		token := r.PostFormValue("token")
		if token == "" {
			response := fmt.Sprintf("{\"error\": \"incorrectToken\"}",)
			writeJsonResponse(response, w)
			return
		}
		var existingBoundValue string
		var boundValueAlreadyExists bool
		err := DatabaseConnection.QueryRow("SELECT userid FROM Users WHERE userid=$1", bindid).Scan(&existingBoundValue)
		if err == nil {
			boundValueAlreadyExists = true
		} else {
			if err == sql.ErrNoRows {
				boundValueAlreadyExists = false
			}
		}
		var currentUserId string
		err = DatabaseConnection.QueryRow("SELECT userid FROM Users WHERE currenttoken=$1", token).Scan(&currentUserId)
		if !boundValueAlreadyExists {
			_, err = DatabaseConnection.Query("UPDATE Users SET userid=$1 WHERE userid=$2", bindid, currentUserId)
			handleError(err, w)
			if (err == nil) {
				response := fmt.Sprintf("{\"result\": \"true\"}",)
				writeJsonResponse(response, w)
				return
			}
		}

		currentBonuses, currentBonusCounts := collectBonusesForUserId(w, currentUserId)
		bonuses4bind, bonusCounts4bind := collectBonusesForUserId(w, bindid)
		existsBonuses := getExists(currentBonuses, bonuses4bind)
		notexistsBonuses := getNotExists(currentBonuses, bonuses4bind)

		for i, existsBonus := range existsBonuses {
			for j, existsBonus4bind := range bonuses4bind {
				if existsBonus4bind == existsBonus {
					newCount := currentBonusCounts[i] + bonusCounts4bind[j]
					_, err = DatabaseConnection.Query("UPDATE Bonuses SET bonusCount=$1 WHERE userid=$2 AND bonusType=$3", newCount, bindid, existsBonus)
					handleError(err, w)
					_, err = DatabaseConnection.Query("DELETE FROM Bonuses WHERE userid=$1 AND bonusType=$2", currentUserId, existsBonus)
					handleError(err, w)
				}
			}
		}

		for _, notexistsbonus := range notexistsBonuses {
			_, err = DatabaseConnection.Query("UPDATE Bonuses SET userid=$1 WHERE userid=$2 AND bonusType=$3", bindid, currentUserId, notexistsbonus)
			handleError(err, w)
		}
		conductorSkins, tramsSkins, events := collectValuesForUserId(w, currentUserId, true)
		conductorSkins4bind, tramsSkins4bind, events4bind := collectValuesForUserId(w, bindid, false)
		existsConductorSkins := getExists(conductorSkins, conductorSkins4bind)
		existsTramSkins := getExists(tramsSkins, tramsSkins4bind)
		existsEvents := getExists(events, events4bind)
		notExistsEvents := getNotExists(events, events4bind)
		for _, notExistsEvent := range notExistsEvents {
			_, err = DatabaseConnection.Query("UPDATE Events SET userid=$1 WHERE userId=$2 AND eventType=$3", bindid, currentUserId, notExistsEvent)
			handleError(err, w)
		}

		for _, existsEvent := range existsEvents {
			var currentStringValue string
			var currentNumberValue int
			var currentFibValue int
			err = DatabaseConnection.QueryRow("SELECT eventNumberValue, eventStringValue, prevFibValue FROM Events WHERE userid=$1 AND eventType=$2", currentUserId, existsEvent).Scan(&currentNumberValue, &currentStringValue, &currentFibValue)
			handleError(err, w)
			var bindStringValue string
			var bindNumberValue int
			var bindFibValue int
			err = DatabaseConnection.QueryRow("SELECT eventNumberValue, eventStringValue, prevFibValue FROM Events WHERE userid=$1 AND eventType=$2", bindid, existsEvent).Scan(&bindNumberValue, &bindStringValue, &bindFibValue)
			handleError(err, w)
			if(currentStringValue == bindStringValue) {
				var maxNumValue int
				maxNumValue = int(math.Max(float64(bindNumberValue), float64(currentNumberValue)))
				var resultFibValue int
				if maxNumValue == bindNumberValue {
					resultFibValue = bindFibValue
				} else {
					resultFibValue = currentFibValue
				}
				_, err = DatabaseConnection.Query("DELETE FROM Events WHERE userid=$1 AND eventStringValue=$2", bindid, bindStringValue)
				handleError(err, w)
				_, err = DatabaseConnection.Query("INSERT INTO Events (eventType, userid, eventNumberValue, eventStringValue, prevFibValue) VALUES ($1, $2, $3, $4, $5)", existsEvent, bindid, maxNumValue, bindStringValue, resultFibValue)
				handleError(err, w)
				_, err = DatabaseConnection.Query("DELETE FROM Events WHERE userid=$1 AND eventStringValue=$2", currentUserId, bindStringValue)
				handleError(err, w)
			} else {
				_, err = DatabaseConnection.Query("UPDATE Events SET userid=$1 WHERE userId=$2 AND eventStringValue=$3", bindid, currentUserId, currentStringValue)
				handleError(err, w)
			}
		}
		for _, existsTramSkin := range existsTramSkins {
			_, err = DatabaseConnection.Query("DELETE FROM TramSkins WHERE userid=$1 AND tramType=$2", currentUserId, existsTramSkin)
			handleError(err, w)
		}
		_, err = DatabaseConnection.Query("UPDATE TramSkins SET userid=$1 WHERE userId=$2", bindid, currentUserId)
		handleError(err, w)

		for _, existsConductorSkin := range existsConductorSkins {
			_, err = DatabaseConnection.Query("DELETE FROM ConductorSkins WHERE userid=$1 AND conductorType=$2", currentUserId, existsConductorSkin)
			handleError(err, w)
		}
		_, err = DatabaseConnection.Query("UPDATE ConductorSkins SET userid=$1 WHERE userId=$2", bindid, currentUserId)
		handleError(err, w)

		var currentGemsCount, currentMoneyCount int
		err = DatabaseConnection.QueryRow("SELECT gemsCount, moneyCount FROM Users WHERE userid=$1", currentUserId).Scan(&currentGemsCount, &currentMoneyCount)
		handleError(err, w)
		var newGemsCount, newMoneyCount int
		err = DatabaseConnection.QueryRow("SELECT gemsCount, moneyCount FROM Users WHERE userid=$1", bindid).Scan(&newGemsCount, &newMoneyCount)
		handleError(err, w)
		newGemsCount += currentGemsCount
		newMoneyCount += currentMoneyCount
		_, err = DatabaseConnection.Query("UPDATE Users SET gemsCount=$1 WHERE userid=$2", newGemsCount, bindid)
		handleError(err, w)
		_, err = DatabaseConnection.Query("UPDATE Users SET moneyCount=$1 WHERE userid=$2", newMoneyCount, bindid)
		handleError(err, w)
		_, err = DatabaseConnection.Query("UPDATE Users SET currentToken=$1 WHERE userid=$2", token, bindid)
		handleError(err, w)
		_, err = DatabaseConnection.Query("DELETE FROM Users WHERE userid=$1", currentUserId)
		handleError(err, w)
		response := fmt.Sprintf("{\"result\": \"true\"}",)
		writeJsonResponse(response, w)
		return
	}
	response := fmt.Sprintf("{\"error\": \"incorrect action\"}",)
	writeJsonResponse(response, w)
}

func eventsList(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	userid := useridFromGetRequest(w, params)
	if userid == "" {
		return
	}
	eventRows, err := DatabaseConnection.Query("SELECT eventType, eventNumberValue, eventStringValue FROM Events WHERE userid=$1", userid)
	if err != sql.ErrNoRows {
		handleError(err, w)
	} else {
		if err != nil {
			response := fmt.Sprintf("{\"error\": \"no events\"}",)
			writeJsonResponse(response, w)
			return
		}
	}
	var result []Event
	defer eventRows.Close()
	for eventRows.Next() {
		currentEvent := Event{}
		err := eventRows.Scan(&currentEvent.EventType, &currentEvent.EventNumberValue, &currentEvent.EventStringValue)
		handleError(err, w)
		result = append(result, currentEvent)
	}
	jsonBytes, err := json.Marshal(result)
	handleError(err, w)
	fmt.Println(string(jsonBytes))
	writeJsonResponse(string(jsonBytes), w)
}

func bonusesList(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	userid := useridFromGetRequest(w, params)
	if userid == "" {
		return
	}
	bonusRows, err := DatabaseConnection.Query("SELECT bonusType, bonusCount FROM Bonuses WHERE userid=$1", userid)
	if err != sql.ErrNoRows {
		handleError(err, w)
	} else {
		if err != nil {
			response := fmt.Sprintf("{\"error\": \"no bonuses\"}")
			writeJsonResponse(response, w)
			return
		}
	}
	var result []Bonus
	defer bonusRows.Close()
	for bonusRows.Next() {
		currentBonus := Bonus{}
		err := bonusRows.Scan(&currentBonus.BonusType, &currentBonus.Count)
		handleError(err, w)
		result = append(result, currentBonus)
	}
	jsonBytes, err := json.Marshal(result)
	handleError(err, w)
	fmt.Println(string(jsonBytes))
	writeJsonResponse(string(jsonBytes), w)
}

func resourcesList(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	userid := useridFromGetRequest(w, params)
	if userid == "" {
		return
	}
	var gemsCount string
	var moneyCount string
	err := DatabaseConnection.QueryRow("SELECT gemsCount, moneyCount FROM Users WHERE userid=$1", userid).Scan(&gemsCount, &moneyCount)
	if err != sql.ErrNoRows {
		handleError(err, w)
	} else {
		if err != nil {
			response := fmt.Sprintf("{\"error\": \"incorrect userid\"}")
			writeJsonResponse(response, w)
			return
		}
	}
	response := fmt.Sprintf("{\"result\": \"true\", \"gemsCount\": %s, \"moneyCount\": %s}",gemsCount, moneyCount)
	writeJsonResponse(response, w)
}

func useridFromGetRequest(w http.ResponseWriter, params httprouter.Params) string {
	userid := params.ByName("userid")
	if userid == "" {
		response := fmt.Sprintf("{\"error\": \"incorrect userid\"}")
		writeJsonResponse(response, w)
		return ""
	}
	return userid
}

func getExists(arr []string, criteria []string) ([]string) {
	var exists []string
	for _, currentString := range arr {
		if contains(criteria, currentString) {
			exists = append(exists, currentString)
		}
	}
	return exists
}

func getNotExists(arr []string, criteria []string) ([]string) {
	var notexists []string
	for _, currentString := range arr {
		if !contains(criteria, currentString) {
			notexists = append(notexists, currentString)
		}
	}
	return notexists
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func collectBonusesForUserId(w http.ResponseWriter, userId string) ([]string, []int) {
	var bonuses []string
	var counts []int
	bonusesRows, err := DatabaseConnection.Query("SELECT bonusType, bonusCount FROM Bonuses WHERE userid=$1", userId)
	if err != sql.ErrNoRows {
		handleError(err, w)
	}
	defer bonusesRows.Close()
	for bonusesRows.Next() {
		var bonusName string
		var currentCount int
		err := bonusesRows.Scan(&bonusName, &currentCount)
		handleError(err, w)
		bonuses = append(bonuses, bonusName)
		counts = append(counts, currentCount)
	}
	return bonuses, counts
}

func collectValuesForUserId(w http.ResponseWriter, userid string, eraseActive bool) ([]string, []string, []string)  {

	var conductorSkins []string
	conductorRows, err := DatabaseConnection.Query("SELECT conductorType, isCurrent FROM ConductorSkins WHERE userid=$1", userid)
	if err != sql.ErrNoRows {
		handleError(err, w)
	}
	defer conductorRows.Close()
	for conductorRows.Next() {
		var conductorName string
		var isCurrent bool
		err := conductorRows.Scan(&conductorName, &isCurrent)
		handleError(err, w)
		conductorSkins = append(conductorSkins, conductorName)
		if isCurrent && eraseActive {
			_, err = DatabaseConnection.Query("UPDATE ConductorSkins SET isCurrent=FALSE WHERE userid=$1 AND conductorType=$2", userid, conductorName)
			handleError(err, w)
		}
	}

	var currenttramSkins []string
	tramRows, err := DatabaseConnection.Query("SELECT tramType, isCurrent FROM TramSkins WHERE userid=$1", userid)
	if err != sql.ErrNoRows {
		handleError(err, w)
	}
	defer tramRows.Close()
	for tramRows.Next() {
		var tramName string
		var isCurrent bool
		err := tramRows.Scan(&tramName, &isCurrent)
		handleError(err, w)
		currenttramSkins = append(currenttramSkins, tramName)
		if isCurrent && eraseActive {
			_, err = DatabaseConnection.Query("UPDATE TramSkins SET isCurrent=FALSE WHERE userid=$1 AND tramType=$2", userid, tramName)
			handleError(err, w)
		}
	}
	var events []string
	eventRows, err := DatabaseConnection.Query("SELECT eventType FROM Events WHERE userid=$1", userid)
	if err != sql.ErrNoRows {
		handleError(err, w)
	}
	defer eventRows.Close()
	for eventRows.Next() {
		var eventName string
		err := eventRows.Scan(&eventName)
		handleError(err, w)
		events = append(events, eventName)
	}
	return conductorSkins, currenttramSkins, events
}

func createUUID() string {
	if Random == nil {
		f, err := os.Open("/dev/urandom")
		if err != nil {
			log.Fatal(err)
		}
		Random = f
	}
	b := make([]byte, 16)
	Random.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}