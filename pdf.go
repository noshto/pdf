package pdf

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/johnfercher/maroto/pkg/color"
	"github.com/johnfercher/maroto/pkg/consts"
	"github.com/johnfercher/maroto/pkg/pdf"
	"github.com/johnfercher/maroto/pkg/props"
	"github.com/noshto/pdf/pkg/xmlutils"
	"github.com/noshto/sep"
	"github.com/skip2/go-qrcode"
)

// Params represents collection of parameters needed for Generate function
type Params struct {
	SepConfig      *sep.Config
	InternalInvNum string
	ReqFile        string
	RespFile       string
	OutFile        string
}

// GeneratePDF generated PDF of invoice from given parameters
func GeneratePDF(params *Params) error {

	request, err := RegisterInvoiceRequest(params.ReqFile)
	if err != nil {
		return err
	}
	response, err := RegisterInvoiceResponse(params.RespFile)
	if err != nil {
		return err
	}

	m := pdf.NewMaroto(consts.Portrait, consts.A4)
	m.SetPageMargins(10, 15, 10)
	m.SetBackgroundColor(color.NewWhite())

	// Some guides:
	// Col full width is 12

	// Company name

	TitleTextAttrib := props.Text{
		Size:        20,
		Align:       consts.Left,
		Style:       consts.Bold,
		Extrapolate: false,
	}

	BodyTextAttrib := props.Text{
		Size:        8,
		Align:       consts.Left,
		Style:       consts.Normal,
		Extrapolate: true,
	}

	BodyRightAlignTextAttrib := props.Text{
		Size:        8,
		Align:       consts.Right,
		Style:       consts.Normal,
		Extrapolate: true,
	}

	CaptionTextAttrib := props.Text{
		Size:        8,
		Align:       consts.Left,
		Style:       consts.Bold,
		Extrapolate: true,
	}

	CaptionRightAlignTextAttrib := props.Text{
		Size:        8,
		Align:       consts.Right,
		Style:       consts.Bold,
		Extrapolate: true,
	}

	buf, err := GenerateQRCode(params.ReqFile, params.SepConfig.Environment)
	if err != nil {
		return err
	}
	b64Image := base64.StdEncoding.EncodeToString(buf)

	// Company Name along with QR code
	m.Row(32, func() {
		m.Col(10, func() {
			m.Text(request.Invoice.Seller.Name, TitleTextAttrib)
		})

		m.Col(2, func() {

			m.Base64Image(b64Image, consts.Png, props.Rect{
				Left:    0,
				Top:     0,
				Percent: 100,
				Center:  true,
			})
			// m.FileImage(qrCodeFile, props.Rect{
			// 	Left:    0,
			// 	Top:     0,
			// 	Percent: 100,
			// 	Center:  true,
			// })
		})

		// Adds extra space from Company name to next lines
		m.Row(20, func() {})

		// Company details
		m.Row(4, func() {
			m.Col(6, func() {
				m.Text(request.Invoice.Seller.Address, BodyTextAttrib)
			})

			m.Col(6, func() {
				m.Text(strings.Join([]string{"PIB:", request.Invoice.Seller.IDNum}, " "), BodyTextAttrib)
			})
		})
		m.Row(4, func() {
			m.Col(6, func() {
				m.Text(strings.Join([]string{"Tel:", params.SepConfig.Phone}, " "), BodyTextAttrib)
			})
			m.Col(6, func() {
				m.Text(strings.Join([]string{"PDV:", params.SepConfig.VAT}, " "), BodyTextAttrib)
			})
		})
		m.Row(4, func() {
			m.Col(6, func() {
				m.Text(strings.Join([]string{"Fax:", params.SepConfig.Fax}, " "), BodyTextAttrib)
			})
			m.Col(6, func() {
				m.Text(strings.Join([]string{"Z.R.:", params.SepConfig.BankAccount}, " "), BodyTextAttrib)
			})
		})
	})

	// Invoice number on the left with Buyer header on the right
	m.Row(6, func() {
		m.Col(6, func() {
			m.Text(strings.Join([]string{"Broj računa:", string(request.Invoice.InvNum)}, " "), CaptionTextAttrib)
			m.Text(strings.Join([]string{"Interni br:", string(params.InternalInvNum)}, " "), CaptionTextAttrib)
		})
		m.Col(6, func() {
			m.Text("KUPAC", CaptionTextAttrib)
		})
	})

	// Invoice issue date on the left and Buyer name on the right
	m.Row(4, func() {
		m.Col(6, func() {
			IssueDateTime := time.Time(request.Invoice.IssueDateTime).Format("2006-01-02")
			m.Text(strings.Join([]string{"Datum prometa dobara:", IssueDateTime}, " "), BodyTextAttrib)
		})
		m.Col(6, func() {
			m.Text(request.Invoice.Buyer.Name, CaptionTextAttrib)
		})
	})

	// Currency on the left and Buyer address on the right
	m.Row(4, func() {
		m.Col(6, func() {
			Currency := "EUR"
			if request.Invoice.Currency != nil {
				Currency = string(request.Invoice.Currency.Code)
			}
			m.Text(strings.Join([]string{"Valuta:", string(Currency)}, " "), BodyTextAttrib)
		})

		addressString := strings.Join([]string{request.Invoice.Buyer.Address, request.Invoice.Buyer.Town, request.Invoice.Buyer.Country}, ", ")
		addressString = strings.TrimRight(addressString, ", ")
		m.Col(6, func() {
			m.Text(strings.Join([]string{"Adresa:", addressString}, " "), BodyTextAttrib)
		})
	})
	// Payment type on the left and Buyer TIN on the right
	m.Row(4, func() {
		m.Col(6, func() {
			var TypeOfInv string
			switch request.Invoice.TypeOfInv {
			case sep.CASH:
				TypeOfInv = "Gotovinski"
			case sep.NONCASH:
				TypeOfInv = "Bezgotovinski"
			}
			m.Text(strings.Join([]string{"Nacin placanja:", TypeOfInv}, " "), BodyTextAttrib)
		})
		m.Col(6, func() {
			m.Text(strings.Join([]string{"PIB:", request.Invoice.Buyer.IDNum}, " "), BodyTextAttrib)
		})
	})
	// Spece on the left and Buyer's VAT on the right
	m.Row(4, func() {
		m.ColSpace(6)
		m.Col(6, func() {
			client := &sep.Client{}
			for _, it := range params.SepConfig.Clients {
				if it.TIN == request.Invoice.Buyer.IDNum {
					client = &it
					break
				}
			}
			m.Text(strings.Join([]string{"PDV:", client.VAT}, " "), BodyTextAttrib)
		})
	})

	m.Line(4)

	content := [][]string{}
	for in, it := range *request.Invoice.Items {
		content = append(
			content,
			[]string{
				strconv.Itoa(in + 1),
				it.N,
				it.U,
				strconv.FormatFloat(float64(it.Q), 'f', 2, 64),
				strconv.FormatFloat(float64(it.UPB), 'f', 2, 64),
				strings.Join([]string{strconv.FormatFloat(float64(it.R), 'f', 2, 64), "%"}, " "),
				strings.Join([]string{strconv.FormatFloat(float64(it.VR), 'f', 2, 64), "%"}, " "),
				strconv.FormatFloat(float64(it.PA), 'f', 2, 64),
			},
		)
	}

	m.TableList(
		[]string{"#", "NAZIV PROIZVODA/USLUGE", "JM", "Kolicina", "Cijena bez PDV", "Rabat %", "PDV stopa", "Cijena sa PDV"},
		content,
		props.TableList{
			HeaderProp: props.TableListContent{
				Size:      9,
				Style:     consts.Bold,
				GridSizes: []uint{1, 3, 1, 1, 2, 1, 1, 2},
			},
			ContentProp: props.TableListContent{
				Size:      8,
				GridSizes: []uint{1, 3, 1, 1, 2, 1, 1, 2},
			},
			Align: consts.Center,
			AlternatedBackground: &color.Color{
				Red:   200,
				Green: 200,
				Blue:  200,
			},
			HeaderContentSpace: 2,
			Line:               false,
		},
	)

	m.Line(0)

	m.Row(4, func() {})

	TotPriceWoVAT := strconv.FormatFloat(float64(request.Invoice.TotPriceWoVAT), 'f', 2, 64)
	TotPrice := strconv.FormatFloat(float64(request.Invoice.TotPrice), 'f', 2, 64)
	TotVATAmt := strconv.FormatFloat(float64(request.Invoice.TotVATAmt), 'f', 2, 64)

	VATRate := 21.0
	for _, it := range *request.Invoice.SameTaxes {
		VATRate = float64(it.VATRate)
	}

	// Invoice summary
	m.Row(4, func() {
		m.ColSpace(5)
		m.Col(3, func() {
			m.Text("Osnovica za stopu 21%:", BodyRightAlignTextAttrib)
		})
		m.Col(4, func() {
			m.Text(TotPriceWoVAT, BodyRightAlignTextAttrib)
		})
	})
	m.Row(4, func() {
		m.ColSpace(5)
		m.Col(3, func() {
			m.Text("Iznos rabata:", BodyRightAlignTextAttrib)
		})
		m.Col(4, func() {
			m.Text("0.00", BodyRightAlignTextAttrib)
		})
	})

	m.Row(4, func() {
		m.ColSpace(5)
		m.Col(3, func() {
			m.Text("Vrijednost bez PDV:", BodyRightAlignTextAttrib)
		})
		m.Col(4, func() {
			m.Text(TotPriceWoVAT, BodyRightAlignTextAttrib)
		})
	})

	m.Row(4, func() {
		m.ColSpace(5)
		m.Col(3, func() {
			m.Text("PDV po stopi 21%", BodyRightAlignTextAttrib)
		})
		m.Col(4, func() {
			m.Text(TotVATAmt, BodyRightAlignTextAttrib)
		})
	})

	m.Line(1)

	m.Row(4, func() {
		m.ColSpace(5)
		m.Col(3, func() {
			m.Text("IZNOS ZA UPLATU:", CaptionRightAlignTextAttrib)
		})
		m.Col(4, func() {
			m.Text(TotPrice, CaptionRightAlignTextAttrib)
		})
	})

	// Extra space between Invoice Summary and IKOF and JIKR codes
	m.Row(4, func() {})

	// JIKR
	m.Row(4, func() {
		m.Col(1, func() {
			m.Text("JIKR:", BodyTextAttrib)
		})
		m.Col(11, func() {
			m.Text(response.Body.RegisterInvoiceResponse.FIC, BodyTextAttrib)
		})
	})

	// IKOF
	m.Row(4, func() {
		m.Col(1, func() {
			m.Text("IKOF:", BodyTextAttrib)
		})
		m.Col(11, func() {
			m.Text(request.Invoice.IIC, props.Text{
				Style:       consts.Normal,
				Size:        8,
				Align:       consts.Left,
				Extrapolate: false,
			})
		})
	})

	m.Row(4, func() {})
	m.Row(4, func() {
		m.Col(6, func() {
			m.Text("NAPOMENA:", CaptionTextAttrib)
		})
	})

	if VATRate == 0 {
		m.Row(4, func() {
			m.Text("PDV obracunat po stopi 0% u skladu sa Clanom 17. Zakona o PDV-u", BodyTextAttrib)
		})
	}

	m.Row(4, func() {
		m.Text("U slucaju ne placanja u dogovorenom roku obracunava se zatezna kamata.", BodyTextAttrib)
	})
	m.Row(4, func() {
		m.Text("U slucaju spora nadlezan je Privredni sud Podgorica", BodyTextAttrib)
	})

	m.Row(4, func() {

		m.ColSpace(6)
		m.Col(6, func() {
			m.Text("M.P. _________________________", BodyRightAlignTextAttrib)
		})
	})

	return m.OutputFileAndClose(params.OutFile)
}

// GenerateQRCode generated QR code for given invoice
func GenerateQRCode(filePath string, env sep.EnvironmentType) ([]byte, error) {
	doc := etree.NewDocument()
	err := doc.ReadFromFile(filePath)
	if err != nil {
		return []byte{}, err
	}

	InvOrdNum, err := xmlutils.ValueOfAttribute("//Invoice", "InvOrdNum", doc)
	if err != nil {
		return []byte{}, err
	}
	TCRCode, err := xmlutils.ValueOfAttribute("//Invoice", "TCRCode", doc)
	if err != nil {
		return []byte{}, err
	}
	TotPrice, err := xmlutils.ValueOfAttribute("//Invoice", "TotPrice", doc)
	if err != nil {
		return []byte{}, err
	}
	BusinUnitCode, err := xmlutils.ValueOfAttribute("//Invoice", "BusinUnitCode", doc)
	if err != nil {
		return []byte{}, err
	}
	SoftCode, err := xmlutils.ValueOfAttribute("//Invoice", "SoftCode", doc)
	if err != nil {
		return []byte{}, err
	}
	IIC, err := xmlutils.ValueOfAttribute("//Invoice", "IIC", doc)
	if err != nil {
		return []byte{}, err
	}
	IssueDateTime, err := xmlutils.ValueOfAttribute("//Invoice", "IssueDateTime", doc)
	if err != nil {
		return []byte{}, err
	}
	TIN, err := xmlutils.ValueOfAttribute("//Seller", "IDNum", doc)
	if err != nil {
		return []byte{}, err
	}

	var format string
	switch env {
	case sep.TEST:
		format = sep.TestingVerifyURL
	case sep.PROD:
		format = sep.ProductionVerifyURL
	default:
		return []byte{}, fmt.Errorf("invalid environment")
	}

	link := fmt.Sprintf(
		format,
		IIC,
		TIN,
		IssueDateTime,
		InvOrdNum,
		BusinUnitCode,
		TCRCode,
		SoftCode,
		TotPrice,
	)

	return qrcode.Encode(link, qrcode.Highest, 256)
}

// RegisterInvoiceRequest retrieves RegisterInvoiceRequest from file
func RegisterInvoiceRequest(filePath string) (*sep.RegisterInvoiceRequest, error) {
	doc := etree.NewDocument()
	err := doc.ReadFromFile(filePath)
	if err != nil {
		return nil, err
	}
	elem := doc.FindElement("//RegisterInvoiceRequest")
	if elem == nil {
		return nil, fmt.Errorf("not valid xml. no RegisterInvoiceRequest")
	}

	reqDoc := etree.NewDocument()
	reqDoc.SetRoot(elem.Copy())
	buf, err := reqDoc.WriteToBytes()
	if err != nil {
		return nil, err
	}

	RegisterInvoiceRequest := sep.RegisterInvoiceRequest{}
	err = xml.Unmarshal(buf, &RegisterInvoiceRequest)
	return &RegisterInvoiceRequest, err
}

// RegisterInvoiceResponse retrieves RegisterInvoiceResponse from file
func RegisterInvoiceResponse(filePath string) (*sep.RegisterInvoiceResponse, error) {
	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	RegisterInvoiceResponse := sep.RegisterInvoiceResponse{}
	err = xml.Unmarshal(buf, &RegisterInvoiceResponse)
	return &RegisterInvoiceResponse, err
}
