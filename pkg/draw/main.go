package draw

import (
	"fmt"
	"image/color"
	"math"

	"github.com/fogleman/gg"
)

type Token struct {
    Name   string
    Amount float64
    Color  color.Color
    Icon   string // Placeholder, you can use actual PNG paths here
}

func main() {
    const width = 800
    const height = 600

    dc := gg.NewContext(width, height)

    // Sky background
    dc.SetRGB(0.53, 0.81, 0.98) // skyblue
    dc.Clear()

    // Ground
    dc.SetRGB(0.56, 0.93, 0.56) // lightgreen
    dc.DrawRectangle(0, float64(height/2), float64(width), float64(height/2))
    dc.Fill()

    // Tokens
    tokens := []Token{
        {"PUNKS", 3400, color.RGBA{34, 139, 34, 255}, "🧑‍🎤"},
        {"SKULLY", 1200, color.RGBA{255, 140, 0, 255}, "💀"},
        {"SOCKZ", 800, color.RGBA{128, 0, 128, 255}, "🧦"},
    }

    total := 0.0
    for _, t := range tokens {
        total += t.Amount
    }

    xOffset := 50.0
    maxWidth := float64(width - 100)
    fieldHeight := 140.0
    spacing := 30.0
    currentY := float64(height/2) + 30

    dc.SetRGB(1, 1, 1)
    fontPath := "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"
    if err := dc.LoadFontFace(fontPath, 20); err != nil {
        panic(err)
    }

    for _, token := range tokens {
        proportion := token.Amount / total
        fieldWidth := proportion * maxWidth

        dc.SetColor(token.Color)
        dc.DrawRectangle(xOffset, currentY, fieldWidth, fieldHeight)
        dc.Fill()

        dc.SetRGB(1, 1, 1)
        label := token.Name + " (" + fmt.Sprintf("%.0f", token.Amount) + ")"
        dc.DrawString(label, xOffset+10, currentY+25)

        rows := 4
        cols := int(math.Floor(fieldWidth / 40))
        for r := 0; r < rows; r++ {
            for c := 0; c < cols; c++ {
                cx := xOffset + 10 + float64(c)*30
                cy := currentY + 40 + float64(r)*25
                if cx+20 < xOffset+fieldWidth {
                    dc.DrawString(token.Icon, cx, cy)
                }
            }
        }

        currentY += fieldHeight + spacing
    }

    dc.SavePNG("fimages/dashboard_detailed.png")
}