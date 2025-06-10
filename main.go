package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 1920
	screenHeight = 1080
	orbSize      = 100
)

type Game struct {
	mana            float64     // Mana with decimal precision
	manaPerSec      int64       // Stored as hundredths (e.g. 150 = 1.50/sec)
	orbX            float64
	orbY            float64
	orbClicked      bool
	clickAnimation  int
	generators      []Generator
	tickCounter     int
	animationTime   float64
	rotationAngles  []float64  // Rotation angles for center indicators
	totalMultiplier float64    // Total multiplicative effect
	fontSource      *text.GoTextFaceSource
}

type Generator struct {
	name           string
	cost           float64  // Cost with decimal precision
	speedPerLevel  float64  // Speed increase per level
	level          int      // Generator level (1-100)
	description    string
	timer          int      // Individual timer for this generator
	manaMultiplier float64  // Accumulated mana multiplier
}

func NewGame() *Game {
	// Load font source from embedded font
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	
	g := &Game{
		mana:         0,
		manaPerSec:   0,
		orbX:         screenWidth/2 - orbSize/2,
		orbY:         screenHeight/2 - orbSize/2,
		generators: []Generator{
			{"Mana Crystal", 3.0, 0.1, 5, "Basic mana generation crystal", 0, 1.0},
			{"Arcane Tower", 50.0, 0.08, 0, "Mystical mana channeling tower", 0, 1.0},  
			{"Ley Line Node", 250.0, 0.05, 0, "Powerful magical energy nexus", 0, 1.0},
			{"Elder Artifact", 1000.0, 0.02, 0, "Ancient relic of immense power", 0, 1.0},
		},
		rotationAngles: make([]float64, 4),
		fontSource:     s,
	}
	
	// Calculate initial mana per second using multiplicative system
	g.calculateManaPerSec()
	
	return g
}

// Calculate mana per second using mana multiplier system
func (g *Game) calculateManaPerSec() {
	// Calculate the product of all mana multipliers
	g.totalMultiplier = 1.0
	
	for _, generator := range g.generators {
		g.totalMultiplier *= generator.manaMultiplier
	}
	
	// Convert to mana per second (keep full precision)
	g.manaPerSec = int64(g.totalMultiplier * 100 + 0.5) // Store as hundredths
}

func (g *Game) Update() error {
	// Handle mouse clicks for generators only
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		g.handleGeneratorClicks(x, y)
	}
	
	// Handle orb click animation (visual effect only)
	if g.clickAnimation > 0 {
		g.clickAnimation--
	}
	if g.clickAnimation == 0 {
		g.orbClicked = false
	}
	
	// Update animation time for visual effects
	g.animationTime += 0.016 // Approximately 1/60th of a second
	
	// Update mana production using mana multiplier system
	g.tickCounter++
	if g.tickCounter >= 60 {
		// Recalculate and produce mana every second
		g.calculateManaPerSec()
		if g.totalMultiplier > 0 {
			// Add mana with full precision
			g.mana += g.totalMultiplier
		}
		g.tickCounter = 0
	}
	
	// Update rotation angles and accumulate mana multipliers
	for i := range g.generators {
		if g.generators[i].level > 0 {
			// Calculate individual generator rotation speed (level * speedPerLevel)
			totalSpeed := g.generators[i].speedPerLevel * float64(g.generators[i].level)
			
			// Update rotation angle for visual indicator (speed 1 = 1 rotation per second)
			rotationSpeed := totalSpeed * 2 * math.Pi / 60.0 // radians per tick
			oldAngle := g.rotationAngles[i]
			g.rotationAngles[i] += rotationSpeed
			
			// Check if completed a full rotation (crossed 2π boundary)
			if oldAngle < 2*math.Pi && g.rotationAngles[i] >= 2*math.Pi {
				// Completed a full rotation, add 0.01 to mana multiplier
				g.generators[i].manaMultiplier += 0.01
			}
			
			// Reset angle if it exceeds 2π
			if g.rotationAngles[i] >= 2*math.Pi {
				g.rotationAngles[i] -= 2*math.Pi
			}
		}
	}
	
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Clear screen with dark blue background
	screen.Fill(color.RGBA{25, 25, 50, 255})
	
	// Draw game stats with large font
	manaText := fmt.Sprintf("Mana: %.2f", g.mana)
	op := &text.DrawOptions{}
	op.GeoM.Translate(20, 50)
	op.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 255})
	text.Draw(screen, manaText, &text.GoTextFace{
		Source: g.fontSource,
		Size:   32, // Large font size
	}, op)
	
	// Build multiplier calculation string
	multiplierStr := ""
	for i, generator := range g.generators {
		if i > 0 {
			multiplierStr += " x "
		}
		multiplierStr += fmt.Sprintf("%.2f", generator.manaMultiplier)
	}
	multiplierStr += fmt.Sprintf(" = %.2f/sec", g.totalMultiplier)
	
	op2 := &text.DrawOptions{}
	op2.GeoM.Translate(20, 100)
	op2.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 255})
	text.Draw(screen, multiplierStr, &text.GoTextFace{
		Source: g.fontSource,
		Size:   24, // Medium font size
	}, op2)
	
	// Draw circular generators visualization (now centered)
	g.drawCircularGenerators(screen)
	
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) isMouseOverOrb(x, y float64) bool {
	centerX := g.orbX + orbSize/2
	centerY := g.orbY + orbSize/2
	dx := x - centerX
	dy := y - centerY
	return dx*dx+dy*dy <= (orbSize/2)*(orbSize/2)
}

func (g *Game) drawCircularGenerators(screen *ebiten.Image) {
	centerX := float32(screenWidth / 2)
	centerY := float32(screenHeight / 2)
	
	
	for i, generator := range g.generators {
		// Draw generator info in corners (scaled positions)
		var textX, textY int
		switch i {
		case 0: // Top left
			textX = 30
			textY = 120
		case 1: // Top right
			textX = screenWidth - 400
			textY = 120
		case 2: // Bottom left
			textX = 30
			textY = screenHeight - 200
		case 3: // Bottom right
			textX = screenWidth - 400
			textY = screenHeight - 200
		}
		
		// Calculate current total speed
		currentSpeed := generator.speedPerLevel * float64(generator.level)
		
		// Draw generator info with large font
		nameText := fmt.Sprintf("%s: Lv%d", generator.name, generator.level)
		costText := fmt.Sprintf("Cost: %.2f (+%.2f speed)", generator.cost, generator.speedPerLevel)
		speedText := fmt.Sprintf("Speed: %.2f", currentSpeed)
		multiplierText := fmt.Sprintf("Multiplier: x%.2f", generator.manaMultiplier)
		
		// Name
		op1 := &text.DrawOptions{}
		op1.GeoM.Translate(float64(textX), float64(textY))
		op1.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 255})
		text.Draw(screen, nameText, &text.GoTextFace{
			Source: g.fontSource,
			Size:   28,
		}, op1)
		
		// Cost
		op2 := &text.DrawOptions{}
		op2.GeoM.Translate(float64(textX), float64(textY+40))
		op2.ColorScale.ScaleWithColor(color.RGBA{200, 200, 200, 255})
		text.Draw(screen, costText, &text.GoTextFace{
			Source: g.fontSource,
			Size:   20,
		}, op2)
		
		// Speed
		op3 := &text.DrawOptions{}
		op3.GeoM.Translate(float64(textX), float64(textY+70))
		op3.ColorScale.ScaleWithColor(color.RGBA{200, 200, 200, 255})
		text.Draw(screen, speedText, &text.GoTextFace{
			Source: g.fontSource,
			Size:   20,
		}, op3)
		
		// Multiplier
		op4 := &text.DrawOptions{}
		op4.GeoM.Translate(float64(textX), float64(textY+100))
		op4.ColorScale.ScaleWithColor(color.RGBA{100, 255, 100, 255})
		text.Draw(screen, multiplierText, &text.GoTextFace{
			Source: g.fontSource,
			Size:   20,
		}, op4)
	}
	
	// Draw production status in center
	g.drawCenterProductionStatus(screen, centerX, centerY)
}

func (g *Game) drawCenterProductionStatus(screen *ebiten.Image, centerX, centerY float32) {
	// Draw rotating indicators for each generator (scaled for larger screen)
	for i, generator := range g.generators {
		if generator.level > 0 {
			indicatorRadius := float32(100 + i*50) // Scaled from 40+i*20 to 100+i*50
			
			// Calculate indicator position based on rotation
			angle := float32(g.rotationAngles[i])
			indicatorX := centerX + indicatorRadius*float32(math.Cos(float64(angle)))
			indicatorY := centerY + indicatorRadius*float32(math.Sin(float64(angle)))
			
			// Draw rotating indicator (larger circle)
			colors := []color.RGBA{
				{255, 100, 100, 255}, // Red
				{255, 200, 100, 255}, // Orange  
				{100, 255, 100, 255}, // Green
				{100, 200, 255, 255}, // Blue
			}
			
			// Draw larger indicator with glow effect (scaled)
			glowColor := colors[i]
			glowColor.A = 100
			vector.DrawFilledCircle(screen, indicatorX, indicatorY, 20, glowColor, false) // Glow (scaled from 8 to 20)
			vector.DrawFilledCircle(screen, indicatorX, indicatorY, 12, colors[i], false) // Main dot (scaled from 5 to 12)
			
			// Draw orbit path (faint circle with thicker stroke)
			pathColor := colors[i]
			pathColor.A = 80
			vector.StrokeCircle(screen, centerX, centerY, indicatorRadius, 3, pathColor, false) // Thicker stroke (1 to 3)
		}
	}
}

func (g *Game) drawArcSegment(screen *ebiten.Image, centerX, centerY, radius, thickness, startAngle, endAngle float32, col color.RGBA) {
	segments := 32
	angleStep := (endAngle - startAngle) / float32(segments)
	
	for i := 0; i < segments; i++ {
		angle1 := startAngle + float32(i)*angleStep
		angle2 := startAngle + float32(i+1)*angleStep
		
		// Draw thick stroke segments using StrokeCircle for simplicity
		midAngle := (angle1 + angle2) / 2
		x := centerX + radius*float32(math.Cos(float64(midAngle)))
		y := centerY + radius*float32(math.Sin(float64(midAngle)))
		
		vector.DrawFilledCircle(screen, x, y, thickness/2, col, false)
	}
}

func (g *Game) handleGeneratorClicks(x, y int) {
	// Check corner text area clicks only (scaled click areas)
	for i := range g.generators {
		var textX, textY int
		switch i {
		case 0: // Top left
			textX = 30
			textY = 120
		case 1: // Top right
			textX = screenWidth - 400
			textY = 120
		case 2: // Bottom left
			textX = 30
			textY = screenHeight - 200
		case 3: // Bottom right
			textX = screenWidth - 400
			textY = screenHeight - 200
		}
		
		if x >= textX && x <= textX+370 &&
			y >= textY && y <= textY+130 {
			
			if g.mana >= float64(g.generators[i].cost) {
				// Check if generator can be leveled up (max level 100)
				if g.generators[i].level < 100 {
					g.mana -= float64(g.generators[i].cost)
					g.generators[i].level++
					
					// Speed is automatically calculated as level * speedPerLevel
					// No need to manually add speed increment
					
					// Recalculate mana per second with new multiplicative values
					g.calculateManaPerSec()
					
					// Increase cost for next purchase (different scaling per generator)
					scalingFactor := []float64{1.15, 1.2, 1.2, 1.2}[i] // Mana Crystal has slower scaling
					g.generators[i].cost = g.generators[i].cost * scalingFactor
				}
			}
			break
		}
	}
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Magic Click - Mana Generator")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	
	game := NewGame()
	
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}