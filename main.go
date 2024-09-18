package main

import (
	"crypto/ecdsa"
	"database/sql"
	"log"
	"math/big"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var db *sql.DB
var ethClient *ethclient.Client

func main() {
	var err error
	db, err = sql.Open("mysql", "root:@tcp(localhost:3306)/wallet_db")
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to initialize database, got error %v", err)
	}

	ethClient, err = ethclient.Dial("https://sepolia.infura.io/v3/21057cfb550149b18cd0c0d47550c659")
	if err != nil {
		log.Fatalf("failed to connect to Ethereum client: %v", err)
	}

	app := fiber.New()

	app.Use(logger.New())
	app.Use(recover.New())

	app.Static("/css", "./public/css")
	app.Static("/js", "./public/js")

	app.Get("/create-wallet", createWalletHandler)
	app.Get("/transfer", transferHandler)
	app.Get("/wallet-info", walletInfoHandler)
	app.Post("/api/create-wallet", createWalletAPI)
	app.Post("/api/transfer", transferAPI) // API route for transferring tokens

	log.Fatal(app.Listen(":3000"))
}

func createWalletHandler(c *fiber.Ctx) error {
	return c.SendFile(filepath.Join("templates", "create_wallet.html"))
}
func transferHandler(c *fiber.Ctx) error {
	return c.SendFile(filepath.Join("templates", "transfer.html"))
}

func walletInfoHandler(c *fiber.Ctx) error {
	return c.SendFile(filepath.Join("templates", "wallet_info.html"))
}

func createWalletAPI(c *fiber.Ctx) error {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate wallet"})
	}
	publicKey := privateKey.PublicKey
	walletAddress := crypto.PubkeyToAddress(publicKey).Hex()

	_, err = db.Exec("INSERT INTO user_wallets (user_id, wallet_address, private_key) VALUES (?, ?, ?)", 1, walletAddress, privateKey.D.String())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to store wallet in database"})
	}

	response := map[string]string{
		"wallet_address": walletAddress,
	}

	return c.JSON(response)
}

func transferAPI(c *fiber.Ctx) error {
	type TransferRequest struct {
		Recipient       string `json:"recipient"`
		Amount          string `json:"amount"`
		Token           string `json:"token"`
		ContractAddress string `json:"contractAddress"`
	}

	var req TransferRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Invalid request"})
	}

	privateKeyHex := "73ce02d55110b391d6b11ae2f6aeaddc94cd2e03f24e7b049078a5570198d9fc"
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Failed to parse private key"})
	}

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKey)

	amount := new(big.Int)
	amount.SetString(req.Amount, 10)

	if req.Token == "ETH" {
		nonce, err := ethClient.PendingNonceAt(c.Context(), fromAddress)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to get nonce"})
		}

		tx := types.NewTransaction(
			nonce,
			common.HexToAddress(req.Recipient),
			amount,
			100000,
			big.NewInt(20000000000),
			nil,
		)

		signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, privateKey)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to sign transaction"})
		}

		err = ethClient.SendTransaction(c.Context(), signedTx)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to send transaction"})
		}

		return c.JSON(fiber.Map{"message": "Transaction successful", "txHash": signedTx.Hash().Hex()})
	} else if req.Token == "ERC20" {
		contractAddress := common.HexToAddress(req.ContractAddress)
		tokenABI, err := abi.JSON(strings.NewReader(`[{
			"constant": false,
			"inputs": [
				{
					"name": "_to",
					"type": "address"
				},
				{
					"name": "_value",
					"type": "uint256"
				}
			],
			"name": "transfer",
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			],
			"payable": false,
			"stateMutability": "nonpayable",
			"type": "function"
		}]`))
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to load token ABI"})
		}

		data, err := tokenABI.Pack("transfer", common.HexToAddress(req.Recipient), amount)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to pack transaction data"})
		}

		nonce, err := ethClient.PendingNonceAt(c.Context(), fromAddress)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to get nonce"})
		}

		tx := types.NewTransaction(
			nonce,
			contractAddress,
			big.NewInt(0),
			100000,
			big.NewInt(20000000000),
			data,
		)

		signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, privateKey)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to sign transaction"})
		}

		err = ethClient.SendTransaction(c.Context(), signedTx)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "Failed to send ERC-20 transaction"})
		}

		return c.JSON(fiber.Map{"message": "ERC-20 transaction successful", "txHash": signedTx.Hash().Hex()})
	}

	return c.Status(400).JSON(fiber.Map{"message": "Invalid token type"})
}
