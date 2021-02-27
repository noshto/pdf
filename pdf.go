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
	Clients        *[]sep.Client
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
	m.Row(20, func() {
		m.Col(12, func() {
			m.Text(request.Invoice.Seller.Name, TitleTextAttrib)
		})

		m.Row(12, func() {})
		// m.Col(2, func() {

		// 	m.Base64Image(b64Image, consts.Png, props.Rect{
		// 		Left:    0,
		// 		Top:     0,
		// 		Percent: 100,
		// 		Center:  true,
		// 	})
		// })

		// // Adds extra space from Company name to next lines
		// m.Row(10, func() {})

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

	m.Line(6)

	// Invoice number on the left with Buyer header on the right
	m.Row(4, func() {
		m.Col(6, func() {
			m.Text(strings.Join([]string{"Broj racuna:", string(request.Invoice.InvNum)}, " "), CaptionTextAttrib)
		})
		m.Col(6, func() {
			m.Text("KUPAC", CaptionTextAttrib)
		})
	})
	m.Row(4, func() {
		m.Col(6, func() {
			m.Text(strings.Join([]string{"Interni br:", string(params.InternalInvNum)}, " "), CaptionTextAttrib)
		})
		m.Col(6, func() {
			m.Text(request.Invoice.Buyer.Name, CaptionTextAttrib)
		})
	})

	// Invoice issue date on the left and Buyer name on the right
	m.Row(4, func() {
		m.Col(6, func() {
			IssueDateTime := time.Time(request.Invoice.IssueDateTime).Format("2006-01-02")
			m.Text(strings.Join([]string{"Datum prometa dobara:", IssueDateTime}, " "), BodyTextAttrib)
		})
		addressString := strings.Join([]string{request.Invoice.Buyer.Address, request.Invoice.Buyer.Town, request.Invoice.Buyer.Country}, ", ")
		addressString = strings.TrimRight(addressString, ", ")
		m.Col(6, func() {
			m.Text(strings.Join([]string{"Adresa:", addressString}, " "), BodyTextAttrib)
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
		m.Col(6, func() {
			m.Text(strings.Join([]string{"PIB:", request.Invoice.Buyer.IDNum}, " "), BodyTextAttrib)
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
			client := &sep.Client{}
			for _, it := range *params.Clients {
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

		q := float64(it.Q)
		upb := float64(it.UPB)
		vr := float64(it.VR)
		r := float64(it.R)

		// Calculations
		// upbr is for "Unit Price Before VAT, Rabat applied"
		upbr := (upb - upb*(r/100))
		// pb is for "Price Before VAT"
		pb := upbr * q

		// uva is for "Unit VAT Amount, Rabat applied"
		uva := upbr * (vr / 100)
		// va is for "VAT Amount"
		va := uva * q

		// upa is for "Unit Price After VAT, Rabat applied"
		upa := upbr + uva
		// pa is for "Price After VAT, Rabat applied"
		pa := pb + va

		Name := it.N
		Unit := it.U
		Quantity := strconv.FormatFloat(float64(it.Q), 'f', 2, 64)
		UnitPriceBefVAT := strconv.FormatFloat(upb, 'f', 2, 64)
		PriceBefVAT := strconv.FormatFloat(upb*q, 'f', 2, 64)
		Rebate := fmt.Sprintf("%d%%", int64(r))
		VATRate := fmt.Sprintf("%d%%", int64(vr))
		VATAmount := strconv.FormatFloat(va, 'f', 2, 64)
		UnitPriceAfterVAT := strconv.FormatFloat(upa, 'f', 2, 64)
		PriceAfterVAT := strconv.FormatFloat(pa, 'f', 2, 64)

		content = append(
			content,
			[]string{
				strconv.Itoa(in + 1),
				Name,
				Unit,
				Quantity,
				UnitPriceBefVAT,
				PriceBefVAT,
				Rebate,
				VATRate,
				VATAmount,
				UnitPriceAfterVAT,
				PriceAfterVAT,
			},
		)
	}

	m.TableList(
		[]string{"Rb", "NAZIV PROIZVODA/USLUGE", "JM", "Kolicina", "      Cijena      bez PDV", "    Vrijednost     bez PDV", "Rabat %", "PDV Stopa", "PDV Iznos", "      Cijena       sa PDV", "    Vrijednost    sa PDV"},
		content,
		props.TableList{
			HeaderProp: props.TableListContent{
				Size:      6,
				Style:     consts.Normal,
				GridSizes: []uint{1, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			},
			ContentProp: props.TableListContent{
				Size:      6,
				GridSizes: []uint{1, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1},
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

	PriceBeforeVAT := float64(0)
	Rebate := float64(0)
	VATAmt := float64(0)
	for _, it := range *request.Invoice.Items {
		PriceBeforeVAT += float64(it.UPB * it.Q)
		Rebate += PriceBeforeVAT * (float64(it.R) / 100)
		VATAmt += (PriceBeforeVAT - Rebate) * (float64(it.VR) / 100)
	}
	Base21 := PriceBeforeVAT - Rebate
	TotPrice := strconv.FormatFloat(float64(request.Invoice.TotPrice), 'f', 2, 64)
	if Rebate != 0 {
		Rebate *= -1
	}

	// Invoice summary
	m.Row(32, func() {
		m.Col(2, func() {
			m.Base64Image(b64Image, consts.Png, props.Rect{
				Left:    0,
				Top:     0,
				Percent: 100,
				Center:  true,
			})
		})
		m.ColSpace(4)
		m.Row(4, func() {
			m.ColSpace(6)
			m.Col(3, func() {
				m.Text("Vrijednost bez PDV:", BodyRightAlignTextAttrib)
			})
			m.Col(3, func() {
				m.Text(strconv.FormatFloat(PriceBeforeVAT, 'f', 2, 64), BodyRightAlignTextAttrib)
			})
		})

		m.Row(4, func() {
			m.ColSpace(6)
			m.Col(3, func() {
				m.Text("Iznos rabata:", BodyRightAlignTextAttrib)
			})
			m.Col(3, func() {
				m.Text(strconv.FormatFloat(Rebate, 'f', 2, 64), BodyRightAlignTextAttrib)
			})
		})

		m.Row(4, func() {
			m.ColSpace(6)
			m.Col(6, func() {
				m.Text("-----------------------------------------------------------------------------------", BodyRightAlignTextAttrib)
			})
		})

		m.Row(4, func() {
			m.ColSpace(6)
			m.Col(3, func() {
				m.Text("Osnovica za stopu 21%:", BodyRightAlignTextAttrib)
			})
			m.Col(3, func() {
				m.Text(strconv.FormatFloat(Base21, 'f', 2, 64), BodyRightAlignTextAttrib)
			})
		})
		m.Row(4, func() {
			m.ColSpace(6)
			m.Col(3, func() {
				m.Text("PDV po stopi 21%:", BodyRightAlignTextAttrib)
			})
			m.Col(3, func() {
				m.Text(strconv.FormatFloat(VATAmt, 'f', 2, 64), BodyRightAlignTextAttrib)
			})
		})

		m.Row(4, func() {
			m.ColSpace(6)
			m.Col(6, func() {
				m.Text("-----------------------------------------------------------------------------------", BodyRightAlignTextAttrib)
			})
		})

		m.Row(4, func() {
			m.ColSpace(6)
			m.Col(3, func() {
				m.Text("IZNOS ZA UPLATU:", CaptionRightAlignTextAttrib)
			})
			m.Col(3, func() {
				m.Text(TotPrice, CaptionRightAlignTextAttrib)
			})
		})
	})
	// m.Row(4, func() {
	// 	m.ColSpace(6)
	// 	m.Col(3, func() {
	// 		m.Text("Iznos rabata:", BodyRightAlignTextAttrib)
	// 	})
	// 	m.Col(3, func() {
	// 		m.Text(strconv.FormatFloat(Rebate, 'f', 2, 64), BodyRightAlignTextAttrib)
	// 	})
	// })

	// m.Row(4, func() {
	// 	m.ColSpace(6)
	// 	m.Col(6, func() {
	// 		m.Text("-----------------------------------------------------------------------------------", BodyRightAlignTextAttrib)
	// 	})
	// })

	// m.Row(4, func() {
	// 	m.ColSpace(6)
	// 	m.Col(3, func() {
	// 		m.Text("Osnovica za stopu 21%:", BodyRightAlignTextAttrib)
	// 	})
	// 	m.Col(3, func() {
	// 		m.Text(strconv.FormatFloat(Base21, 'f', 2, 64), BodyRightAlignTextAttrib)
	// 	})
	// })
	// m.Row(4, func() {
	// 	m.ColSpace(6)
	// 	m.Col(3, func() {
	// 		m.Text("PDV po stopi 21%:", BodyRightAlignTextAttrib)
	// 	})
	// 	m.Col(3, func() {
	// 		m.Text(strconv.FormatFloat(VATAmt, 'f', 2, 64), BodyRightAlignTextAttrib)
	// 	})
	// })

	// m.Row(4, func() {
	// 	m.ColSpace(6)
	// 	m.Col(6, func() {
	// 		m.Text("-----------------------------------------------------------------------------------", BodyRightAlignTextAttrib)
	// 	})
	// })

	// m.Row(4, func() {
	// 	m.ColSpace(6)
	// 	m.Col(3, func() {
	// 		m.Text("IZNOS ZA UPLATU:", CaptionRightAlignTextAttrib)
	// 	})
	// 	m.Col(3, func() {
	// 		m.Text(TotPrice, CaptionRightAlignTextAttrib)
	// 	})
	// })

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

	ExText := ""
	for _, it := range *request.Invoice.Items {
		switch it.EX {
		case sep.CL17:
			ExText = "PDV obracunat po stopi 0% u skladu sa Clanom 17. Zakona o PDV-u"
			break
		case sep.CL20:
			ExText = "PDV obracunat po stopi 0% u skladu sa Clanom 20. Zakona o PDV-u"
			break
		case sep.CL26:
			ExText = "PDV obracunat po stopi 0% u skladu sa Clanom 26. Zakona o PDV-u"
			break
		case sep.CL27:
			ExText = "PDV obracunat po stopi 0% u skladu sa Clanom 27. Zakona o PDV-u"
			break
		case sep.CL28:
			ExText = "PDV obracunat po stopi 0% u skladu sa Clanom 28. Zakona o PDV-u"
			break
		case sep.CL29:
			ExText = "PDV obracunat po stopi 0% u skladu sa Clanom 29. Zakona o PDV-u"
			break
		case sep.CL30:
			ExText = "PDV obracunat po stopi 0% u skladu sa Clanom 30. Zakona o PDV-u"
			break
		default:
			continue
		}
	}
	if ExText != "" {
		m.Row(4, func() {
			m.Text(ExText, BodyTextAttrib)
		})
		// extra space
		m.Row(2, func() {
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

// GenerateExempt generates exempt from given data
func GenerateExempt(
	cfg *sep.Config,
	from, to time.Time,
	num int,
	PBWoR, R, PBR, VA, Total float64,
	filePath string,
) error {

	m := pdf.NewMaroto(consts.Portrait, consts.A4)
	m.SetPageMargins(10, 15, 10)
	m.SetBackgroundColor(color.NewWhite())

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

	CaptionTextAttrib := props.Text{
		Size:        8,
		Align:       consts.Left,
		Style:       consts.Bold,
		Extrapolate: true,
	}

	// Company Name
	m.Row(20, func() {
		m.Col(12, func() {
			m.Text(cfg.Name, TitleTextAttrib)
		})

		m.Row(12, func() {})

		// Company details
		m.Row(4, func() {
			m.Col(6, func() {
				m.Text(cfg.Address, BodyTextAttrib)
			})

			m.Col(6, func() {
				m.Text(strings.Join([]string{"PIB:", cfg.TIN}, " "), BodyTextAttrib)
			})
		})
		m.Row(4, func() {
			m.Col(6, func() {
				m.Text(strings.Join([]string{"Tel:", cfg.Phone}, " "), BodyTextAttrib)
			})
			m.Col(6, func() {
				m.Text(strings.Join([]string{"PDV:", cfg.VAT}, " "), BodyTextAttrib)
			})
		})
		m.Row(4, func() {
			m.Col(6, func() {
				m.Text(strings.Join([]string{"Fax:", cfg.Fax}, " "), BodyTextAttrib)
			})
			m.Col(6, func() {
				m.Text(strings.Join([]string{"Z.R.:", cfg.BankAccount}, " "), BodyTextAttrib)
			})
		})
	})

	m.Line(6)

	// Period
	m.Row(4, func() {
		m.Col(6, func() {
			From := time.Time(from).Format("2006-01-02")
			To := time.Time(to).Format("2006-01-02")
			period := strings.Join([]string{From, To}, " - ")
			m.Text(strings.Join([]string{"IZVESTAJ ZA PERIOD:", period}, " "), BodyTextAttrib)
		})
	})

	m.Row(5, func() {})

	// Summary
	m.Row(32, func() {
		m.ColSpace(6)
		m.Row(4, func() {
			m.Col(3, func() {
				m.Text("Koliko ukupno faktura:", BodyTextAttrib)
			})
			m.Col(3, func() {
				m.Text(strconv.FormatInt(int64(num), 10), BodyTextAttrib)
			})
			m.ColSpace(6)
		})

		m.Row(4, func() {
			m.Col(3, func() {
				m.Text("Koliko osnovica prije rabata:", BodyTextAttrib)
			})
			m.Col(3, func() {
				m.Text(strconv.FormatFloat(PBWoR, 'f', 2, 64), BodyTextAttrib)
			})
			m.ColSpace(6)
		})

		m.Row(4, func() {
			m.Col(3, func() {
				m.Text("Koliko rabat:", BodyTextAttrib)
			})
			m.Col(3, func() {
				m.Text(strconv.FormatFloat(R, 'f', 2, 64), BodyTextAttrib)
			})
			m.ColSpace(6)
		})
		m.Row(4, func() {
			m.Col(3, func() {
				m.Text("Koliko osnovica posle rabata:", BodyTextAttrib)
			})
			m.Col(3, func() {
				m.Text(strconv.FormatFloat(PBR, 'f', 2, 64), BodyTextAttrib)
			})
			m.ColSpace(6)
		})

		m.Row(4, func() {
			m.Col(3, func() {
				m.Text("Koliko PDV:", BodyTextAttrib)
			})
			m.Col(3, func() {
				m.Text(strconv.FormatFloat(VA, 'f', 2, 64), BodyTextAttrib)
			})
			m.ColSpace(6)
		})

		m.Row(4, func() {
			m.Col(3, func() {
				m.Text("Koliko ukupno sa PDV:", CaptionTextAttrib)
			})
			m.Col(3, func() {
				m.Text(strconv.FormatFloat(Total, 'f', 2, 64), CaptionTextAttrib)
			})
			m.ColSpace(6)
		})
	})

	return m.OutputFileAndClose(filePath)
}
