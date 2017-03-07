package main

// This is a mortgage chaincode for Mashreq Bank
// Mashreq Bank will Deploy the chaincode
// After verification, Mashreq will Add details of verification (Property Details, Seller Account, Buyer Share)
// RERA will then check the sell condition and approve or reject the request
// Upon approval, it will Add hash of signed documents and links for DMS and then AutoEvents will be called
// Mashreq will Update remaining mortgage amount

import (
	//"encoding/base64"
	"strconv"
	"errors"
	"fmt"
	//"flag"
	"time"
	"encoding/json"
	//"database/sql"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/crypto/primitives"
	"github.com/op/go-logging"
	"math/rand"
	//"github.com/go-mssqldb"
)
 
var myLogger = logging.MustGetLogger("Reconciliation")

type ReconChaincode struct  {
}

var statuses = map[int]string{
  1: "Initiated",
  2: "Recieved",
  3: "Authorized",
  4: "AuthRecieved",
  5: "Reconciled",
  6: "Settled",
} //To access this constant, use -> fmt.Println(statuses [5])
	
	
type ReconciliationStruct struct {
	ReferenceNumber string `json:"ReferenceNumber"`
	BillNumber string `json:"BillNumber"`
	BillingCompany string `json:"BillingCompany"`
	Source string `json:"Source"`
	Amount string `json:"Amount"`
	Status string `json:"Status"`
	BatchID string `json:"BatchID"`
	DateTime []string `json:"DateTime"`
	Details []string `json:"Details"`
}

func (t *ReconChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error)  {
	myLogger.Debug("Init Chaincode...")
	if len(args) != 0 {
		return nil, errors.New("Incorrect number of arguments. Expecting 0")
	}

	err := stub.CreateTable("Reconciliation", []*shim.ColumnDefinition{
		&shim.ColumnDefinition{Name: "ReferenceNumber", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "BillNumber", Type: shim.ColumnDefinition_STRING, Key: false},
		&shim.ColumnDefinition{Name: "BillingCompany", Type: shim.ColumnDefinition_STRING, Key: false},
		&shim.ColumnDefinition{Name: "Source", Type: shim.ColumnDefinition_STRING, Key: false},
		&shim.ColumnDefinition{Name: "Amount", Type: shim.ColumnDefinition_STRING, Key: false},		
		&shim.ColumnDefinition{Name: "Status", Type: shim.ColumnDefinition_STRING, Key: false},
		&shim.ColumnDefinition{Name: "BatchID", Type: shim.ColumnDefinition_STRING, Key: false},
	})
	if err != nil {
		return nil, errors.New("Failed creating Reconciliation table.")
	}
	
	err1 := stub.CreateTable("Transactions", []*shim.ColumnDefinition{
		&shim.ColumnDefinition{Name: "ReferenceNumber", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "DateTime", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "Details", Type: shim.ColumnDefinition_STRING, Key: true},
	})
	if err1 != nil {
		return nil, errors.New("Failed creating Transactions table.")
	}

	myLogger.Debug("Init Chaincode...done")

	return nil, nil
}

func (t *ReconChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error)  {
	if function == "initiateTran" {
		return t.initiateTran(stub, args)
	}	else if function == "gatewayTranLeg1" {
		return t.gatewayTranLeg1(stub, args)
	}	else if function == "networkTran" {
		return t.networkTran(stub, args)
	}	else if function == "gatewayTranLeg2" {
		return t.gatewayTranLeg2(stub, args)
	}	else if function == "reconcileTran" {
		return t.reconcileTran(stub, args)
	}
	return nil, errors.New("Received unknown function invocation")
}

func (t *ReconChaincode) initiateTran(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	myLogger.Debug("initiateTran...")

	if len(args) != 4 {
		return nil, errors.New("Incorrect number of arguments. Expecting 4")
	}

	referenceNumber := strconv.Itoa(rand.Intn(100000))
	billNumber := args[0]
	amount := args[1]
	billingCompany := args[2]
	source := args[3]
	status := "1"
	batchId := ""
	
	ok, err := stub.InsertRow("Reconciliation", shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: referenceNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: billNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: amount}},
			&shim.Column{Value: &shim.Column_String_{String_: billingCompany}},
			&shim.Column{Value: &shim.Column_String_{String_: source}},
			&shim.Column{Value: &shim.Column_String_{String_: status}},
			&shim.Column{Value: &shim.Column_String_{String_: batchId}}},
	})
	
	myLogger.Debug("Recon Insert: ", ok)
	myLogger.Debug("Error: ", err)
	
	timestamp := time.Now()
	
	okay, err := stub.InsertRow("Transactions", shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: referenceNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: timestamp.String()}},
			&shim.Column{Value: &shim.Column_String_{String_: "Transaction Initiated on Ledger"}}},
	})

	myLogger.Debug("Tran Insert: ", okay)
	myLogger.Debug("Error: ", err)
	
	reconStruct := ReconciliationStruct{
		ReferenceNumber: referenceNumber,
		BillNumber: billNumber,
		BillingCompany: billingCompany,
		Source: source,
		Amount: amount,		
		Status: status,
		BatchID: batchId,
		DateTime: []string{timestamp.String()},
		Details: []string{"Transaction Initiated on Ledger"},
	}
	
	reconBytes, err := json.Marshal(reconStruct)
	if err != nil {
		myLogger.Errorf("reconciliation transaction marshaling error %v", err)
	}

	myLogger.Debug("reconStruct before: ", reconStruct)
	
	err = stub.PutState(referenceNumber, reconBytes)
	
	reconStruct1 := &ReconciliationStruct{}
	err1 := json.Unmarshal(reconBytes, reconStruct1)
	
	myLogger.Debug("reconStruct after: ", reconStruct1)
	myLogger.Debug("error: ", err1)

	
	
	myLogger.Debug("Transaction Initiated on Ledger with reference number: .", referenceNumber)
		
		
	return []byte(referenceNumber), nil
}

func (t *ReconChaincode) gatewayTranLeg1(stub shim.ChaincodeStubInterface, args []string) ([]byte, error){
	myLogger.Debug("gatewayTranLeg1...")
	
	if len(args) != 5 {
		return nil, errors.New("Incorrect number of arguments. Expecting 5")
	}

	referenceNumber := args[0]
	billNumber := args[1]
	amount := args[2]
	billingCompany := args[3]
	source := args[4]
	status := "2"
	batchId := ""
	
	
	val, err := t.queryTable(stub, referenceNumber)
	if err != nil {
		return nil, errors.New("Unable to get data from state.")
	}
	
	intStatus, err := strconv.Atoi(val.Status)
	
	if (billNumber != val.BillNumber) ||  (amount != val.Amount) || (source != val.Source) || (billingCompany != val.BillingCompany) || (intStatus > 2) {
		return nil, errors.New("Unable to reconcile record.")
	}
	
	err = stub.DeleteRow(
		"Reconciliation",
		[]shim.Column{shim.Column{Value: &shim.Column_String_{String_: referenceNumber}}},
	)
	if err != nil {
		return nil, errors.New("Failed deliting row.")
	}
	
	abc, err1 := stub.InsertRow("Reconciliation", shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: referenceNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: billNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: amount}},
			&shim.Column{Value: &shim.Column_String_{String_: billingCompany}},
			&shim.Column{Value: &shim.Column_String_{String_: source}},
			&shim.Column{Value: &shim.Column_String_{String_: status}},
			&shim.Column{Value: &shim.Column_String_{String_: batchId}}},
	})
	if err1 != nil {
		return nil, errors.New("Failed inserting row.")
	}
	myLogger.Debug("Recon Insert: ", abc)
	
	timestamp := time.Now()
	
	err = stub.DeleteRow(
		"Transactions",
		[]shim.Column{shim.Column{Value: &shim.Column_String_{String_: referenceNumber}}},
	)
	if err != nil {
		return nil, errors.New("Failed deliting row.")
	}
	
	ok, err := stub.InsertRow("Transactions", shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: referenceNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: timestamp.String()}},
			&shim.Column{Value: &shim.Column_String_{String_: "Transaction Recieved at Gateway"}}},
	})
	myLogger.Debug("Tran Insert: ", ok)
	
	
	reconStruct := ReconciliationStruct{
		ReferenceNumber: referenceNumber,
		BillNumber: billNumber,
		BillingCompany: billingCompany,
		Source: source,
		Amount: amount,		
		Status: status,
		BatchID: batchId,
		DateTime: append(val.DateTime, timestamp.String()),
		Details: append(val.Details, "Transaction Recieved at Gateway"),
	}
	
	reconBytes, err := json.Marshal(reconStruct)
	if err != nil {
		myLogger.Errorf("reconciliation transaction marshaling error %v", err)
	}

	err = stub.PutState(referenceNumber, reconBytes)

	myLogger.Debug("Transaction Recieved at Gateway.")
		
	return nil, nil
}

func (t *ReconChaincode) networkTran(stub shim.ChaincodeStubInterface, args []string) ([]byte, error){
	myLogger.Debug("networkTran...")
	
	if len(args) != 5 {
		return nil, errors.New("Incorrect number of arguments. Expecting 5")
	}

	referenceNumber := args[0]
	billNumber := args[1]
	amount := args[2]
	billingCompany := args[3]
	source := args[4]
	status := "3"
	batchId := ""
	
	val, err := t.queryTable(stub, referenceNumber)
	if err != nil {
		return nil, errors.New("Unable to get data from state.")
	}
	
	intStatus, err := strconv.Atoi(val.Status)
	
	if (billNumber != val.BillNumber) ||  (amount != val.Amount) || (source != val.Source) || (billingCompany != val.BillingCompany) || (intStatus > 3)  {
		return nil, errors.New("Unable to reconcile record.")
	}
	
	err = stub.DeleteRow(
		"Reconciliation",
		[]shim.Column{shim.Column{Value: &shim.Column_String_{String_: referenceNumber}}},
	)
	if err != nil {
		return nil, errors.New("Failed deliting row.")
	}
	
	abc, err1 := stub.InsertRow("Reconciliation", shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: referenceNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: billNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: amount}},
			&shim.Column{Value: &shim.Column_String_{String_: billingCompany}},
			&shim.Column{Value: &shim.Column_String_{String_: source}},
			&shim.Column{Value: &shim.Column_String_{String_: status}},
			&shim.Column{Value: &shim.Column_String_{String_: batchId}}},
	})
	if err1 != nil {
		return nil, errors.New("Failed inserting row.")
	}
	myLogger.Debug("Recon Insert: ", abc)
	
	timestamp := time.Now()
	
	err = stub.DeleteRow(
		"Transactions",
		[]shim.Column{shim.Column{Value: &shim.Column_String_{String_: referenceNumber}}},
	)
	if err != nil {
		return nil, errors.New("Failed deleting row.")
	}
	
	ok, err := stub.InsertRow("Transactions", shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: referenceNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: timestamp.String()}},
			&shim.Column{Value: &shim.Column_String_{String_: "Transaction Recieved at Issuer End"}}},
	})
	myLogger.Debug("Tran Insert: ", ok)
	
	
	reconStruct := ReconciliationStruct{
		ReferenceNumber: referenceNumber,
		BillNumber: billNumber,
		BillingCompany: billingCompany,
		Source: source,
		Amount: amount,		
		Status: status,
		BatchID: batchId,
		DateTime: append(val.DateTime, timestamp.String()),
		Details: append(val.Details, "Transaction Recieved at Issuer End"),
	}
	
	reconBytes, err := json.Marshal(reconStruct)
	if err != nil {
		myLogger.Errorf("reconciliation transaction marshaling error %v", err)
	}

	err = stub.PutState(referenceNumber, reconBytes)

	myLogger.Debug("Transaction Recieved at Issuer End.")
		
	return nil, nil
}

func (t *ReconChaincode) gatewayTranLeg2(stub shim.ChaincodeStubInterface, args []string) ([]byte, error){
	myLogger.Debug("gatewayTranLeg2...")
	
	if len(args) != 5 {
		return nil, errors.New("Incorrect number of arguments. Expecting 5")
	}

	referenceNumber := args[0]
	billNumber := args[1]
	amount := args[2]
	billingCompany := args[3]
	source := args[4]
	status := "4"
	batchId := ""
	
	val, err := t.queryTable(stub, referenceNumber)
	if err != nil {
		return nil, errors.New("Unable to get data from state.")
	}
	
	intStatus, err := strconv.Atoi(val.Status)
	
	if (billNumber != val.BillNumber) ||  (amount != val.Amount) || (source != val.Source) || (billingCompany != val.BillingCompany) || (intStatus > 4)  {
		return nil, errors.New("Unable to reconcile record.")
	}
	
	err = stub.DeleteRow(
		"Reconciliation",
		[]shim.Column{shim.Column{Value: &shim.Column_String_{String_: referenceNumber}}},
	)
	if err != nil {
		return nil, errors.New("Failed deliting row.")
	}
	
	abc, err1 := stub.InsertRow("Reconciliation", shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: referenceNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: billNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: amount}},
			&shim.Column{Value: &shim.Column_String_{String_: billingCompany}},
			&shim.Column{Value: &shim.Column_String_{String_: source}},
			&shim.Column{Value: &shim.Column_String_{String_: status}},
			&shim.Column{Value: &shim.Column_String_{String_: batchId}}},
	})
	if err1 != nil {
		return nil, errors.New("Failed inserting row.")
	}
	myLogger.Debug("Recon Insert: ", abc)
	
	timestamp := time.Now()
	
	err = stub.DeleteRow(
		"Transactions",
		[]shim.Column{shim.Column{Value: &shim.Column_String_{String_: referenceNumber}}},
	)
	if err != nil {
		return nil, errors.New("Failed deleting row.")
	}
	
	ok, err := stub.InsertRow("Transactions", shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: referenceNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: timestamp.String()}},
			&shim.Column{Value: &shim.Column_String_{String_: "Authorized transaction recieved back at gateway"}}},
	})
	myLogger.Debug("Tran Insert: ", ok)
	
	
	reconStruct := ReconciliationStruct{
		ReferenceNumber: referenceNumber,
		BillNumber: billNumber,
		BillingCompany: billingCompany,
		Source: source,
		Amount: amount,		
		Status: status,
		BatchID: batchId,
		DateTime: append(val.DateTime, timestamp.String()),
		Details: append(val.Details, "Authorized transaction recieved back at gateway"),
	}
	
	reconBytes, err := json.Marshal(reconStruct)
	if err != nil {
		myLogger.Errorf("reconciliation transaction marshaling error %v", err)
	}

	err = stub.PutState(referenceNumber, reconBytes)

	myLogger.Debug("Authorized transaction recieved back at gateway.")
		
	return nil, nil
}

func (t *ReconChaincode) reconcileTran(stub shim.ChaincodeStubInterface, args []string) ([]byte, error){
	myLogger.Debug("reconcileTran...")
	
	if len(args) != 5 {
		return nil, errors.New("Incorrect number of arguments. Expecting 5")
	}

	referenceNumber := args[0]
	billNumber := args[1]
	amount := args[2]
	billingCompany := args[3]
	source := args[4]
	status := "5"
	batchId := ""
	
	val, err := t.queryTable(stub, referenceNumber)
	if err != nil {
		return nil, errors.New("Unable to get data from state.")
	}
	
	intStatus, err := strconv.Atoi(val.Status)
	
	if (billNumber != val.BillNumber) ||  (amount != val.Amount) || (source != val.Source) || (billingCompany != val.BillingCompany) || (intStatus > 5)  {
		return nil, errors.New("Unable to reconcile record.")
	}
	
	err = stub.DeleteRow(
		"Reconciliation",
		[]shim.Column{shim.Column{Value: &shim.Column_String_{String_: referenceNumber}}},
	)
	if err != nil {
		return nil, errors.New("Failed deliting row.")
	}
	
	abc, err1 := stub.InsertRow("Reconciliation", shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: referenceNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: billNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: amount}},
			&shim.Column{Value: &shim.Column_String_{String_: billingCompany}},
			&shim.Column{Value: &shim.Column_String_{String_: source}},
			&shim.Column{Value: &shim.Column_String_{String_: status}},
			&shim.Column{Value: &shim.Column_String_{String_: batchId}}},
	})
	if err1 != nil {
		return nil, errors.New("Failed inserting row.")
	}
	myLogger.Debug("Recon Insert: ", abc)
	
	timestamp := time.Now()
	
	err = stub.DeleteRow(
		"Transactions",
		[]shim.Column{shim.Column{Value: &shim.Column_String_{String_: referenceNumber}}},
	)
	if err != nil {
		return nil, errors.New("Failed deleting row.")
	}
	
	ok, err := stub.InsertRow("Transactions", shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: referenceNumber}},
			&shim.Column{Value: &shim.Column_String_{String_: timestamp.String()}},
			&shim.Column{Value: &shim.Column_String_{String_: "Transaction reconciled"}}},
	})
	myLogger.Debug("Tran Insert: ", ok)
	
	
	reconStruct := ReconciliationStruct{
		ReferenceNumber: referenceNumber,
		BillNumber: billNumber,
		BillingCompany: billingCompany,
		Source: source,
		Amount: amount,		
		Status: status,
		BatchID: batchId,
		DateTime: append(val.DateTime, timestamp.String()),
		Details: append(val.Details, "Transaction reconciled"),
	}
	
	reconBytes, err := json.Marshal(reconStruct)
	if err != nil {
		myLogger.Errorf("reconciliation transaction marshaling error %v", err)
	}

	err = stub.PutState(referenceNumber, reconBytes)

	myLogger.Debug("Transaction reconciled.")
		
	return nil, nil
}

func (t *ReconChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error)  {
	if function == "GetTranStatus" {
		return t.GetTranStatus(stub, args)
	} else if function == "GetTranData" {
		return t.GetTranData(stub, args)
	}
	
	return nil, errors.New("Received unknown function invocation")
}

func (t *ReconChaincode) GetTranStatus(stub shim.ChaincodeStubInterface, args []string) ([]byte, error){
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	referenceNum := args[0]
	
	val, err := t.queryTable(stub, referenceNum)
	if err != nil {
		myLogger.Debugf("Unable to get data from state")
	}

	myLogger.Debug("referenceNumber: ", val.ReferenceNumber)
	myLogger.Debug("billNumber: ", val.BillNumber)
	myLogger.Debug("source: ", val.Source)
	myLogger.Debug("amount: ", val.Amount)
	myLogger.Debug("billingCompany: ", val.BillingCompany)
	myLogger.Debug("status: ", val.Status)
	myLogger.Debug("batchId: ", val.BatchID)
	
	myLogger.Debug("val.Status", val.Status)
	
	return []byte(val.Status), nil
}

func (t *ReconChaincode) GetTranData(stub shim.ChaincodeStubInterface, args []string) ([]byte, error){
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	referenceNum := args[0]

	val, err := t.queryTable(stub, referenceNum)
	if err != nil {
		myLogger.Debugf("Unable to get data from state")
	}	
	
	reconStruct := ReconciliationStruct{
		ReferenceNumber: val.ReferenceNumber,
		BillNumber: val.BillNumber,
		BillingCompany: val.BillingCompany,
		Source: val.Source,
		Amount: val.Amount,		
		Status: val.Status,
		BatchID: val.BatchID,
		DateTime: val.DateTime,
		Details: val.Details,
	}
	
	reconBytes, err := json.Marshal(reconStruct)
	if err != nil {
		myLogger.Errorf("reconciliation transaction marshaling error %v", err)
	}
	
	return reconBytes, nil
}

func (t *ReconChaincode) queryTable(stub shim.ChaincodeStubInterface, referenceNum string) (*ReconciliationStruct, error) {

	reconBytes, err := stub.GetState(referenceNum)
	if err != nil {
		return nil, errors.New("Failed to get Reconciliation Transaction." + err.Error())
	}
	if len(reconBytes) == 0 {
		return nil, fmt.Errorf("recon transaction %s not exists.", referenceNum)
	}

	reconStruct := &ReconciliationStruct{}
	err = json.Unmarshal(reconBytes, reconStruct)
	if err != nil {
		return nil, errors.New("Failed to parse reconciliation Info. " + err.Error())
	}
	return reconStruct, nil
}

func main() {
	
	primitives.SetSecurityLevel("SHA3", 256)
	err := shim.Start(new(ReconChaincode))
	if err != nil {
		fmt.Printf("Error starting Chaincode: %s", err)
	}
}
