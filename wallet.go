package goar

import (
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"math/big"

	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"github.com/everFinance/gojwk"
)

type Wallet struct {
	Client  *Client
	PubKey  *rsa.PublicKey
	PrvKey  *rsa.PrivateKey
	Address string
}

// proxyUrl: option
func NewWalletFromPath(path string, clientUrl string, proxyUrl ...string) (*Wallet, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return NewWallet(b, clientUrl, proxyUrl...)
}

func NewWallet(b []byte, clientUrl string, proxyUrl ...string) (w *Wallet, err error) {
	key, err := gojwk.Unmarshal(b)
	if err != nil {
		return
	}

	pubKey, err := key.DecodePublicKey()
	if err != nil {
		return
	}
	pub, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		err = fmt.Errorf("pubKey type error")
		return
	}
	prvKey, err := key.DecodePrivateKey()
	if err != nil {
		return
	}
	prv, ok := prvKey.(*rsa.PrivateKey)
	if !ok {
		err = fmt.Errorf("prvKey type error")
		return
	}

	addr := sha256.Sum256(pub.N.Bytes())
	w = &Wallet{
		Client:  NewClient(clientUrl, proxyUrl...),
		PubKey:  pub,
		PrvKey:  prv,
		Address: utils.Base64Encode(addr[:]),
	}

	return
}

func (w *Wallet) Owner() string {
	return utils.Base64Encode(w.PubKey.N.Bytes())
}

func (w *Wallet) SendAR(amount *big.Float, target string, tags []types.Tag) (id string, err error) {
	return w.SendWinstonSpeedUp(utils.ARToWinston(amount), target, tags, 0)
}

func (w *Wallet) SendARSpeedUp(amount *big.Float, target string, tags []types.Tag, speedFactor int64) (id string, err error) {
	return w.SendWinstonSpeedUp(utils.ARToWinston(amount), target, tags, speedFactor)
}

func (w *Wallet) SendWinston(amount *big.Int, target string, tags []types.Tag) (id string, err error) {
	return w.SendWinstonSpeedUp(amount, target, tags, 0)
}

func (w *Wallet) SendWinstonSpeedUp(amount *big.Int, target string, tags []types.Tag, speedFactor int64) (id string, err error) {
	reward, err := w.Client.GetTransactionPrice(nil, &target)
	if err != nil {
		return
	}

	tx := &types.Transaction{
		Format:   2,
		Target:   target,
		Quantity: amount.String(),
		Tags:     utils.TagsEncode(tags),
		Data:     "",
		DataSize: "0",
		Reward:   fmt.Sprintf("%d", reward*(100+speedFactor)/100),
	}

	return w.SendTransaction(tx)
}

func (w *Wallet) SendData(data []byte, tags []types.Tag) (id string, err error) {
	return w.SendDataSpeedUp(data, tags, 0)
}

// SendDataSpeedUp set speedFactor for speed up
// eg: speedFactor = 10, reward = 1.1 * reward
func (w *Wallet) SendDataSpeedUp(data []byte, tags []types.Tag, speedFactor int64) (id string, err error) {
	reward, err := w.Client.GetTransactionPrice(data, nil)
	if err != nil {
		return
	}

	tx := &types.Transaction{
		Format:   2,
		Target:   "",
		Quantity: "0",
		Tags:     utils.TagsEncode(tags),
		Data:     utils.Base64Encode(data),
		DataSize: fmt.Sprintf("%d", len(data)),
		Reward:   fmt.Sprintf("%d", reward*(100+speedFactor)/100),
	}

	return w.SendTransaction(tx)
}

// SendTransaction: if send success, should return pending
func (w *Wallet) SendTransaction(tx *types.Transaction) (id string, err error) {
	anchor, err := w.Client.GetTransactionAnchor()
	if err != nil {
		return
	}
	tx.LastTx = anchor
	tx.Owner = w.Owner()
	if err = utils.SignTransaction(tx, w.PrvKey); err != nil {
		return
	}

	id = tx.ID

	uploader, err := CreateUploader(w.Client, tx, nil)
	if err != nil {
		return
	}
	err = uploader.Once()
	return
}
