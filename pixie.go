package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/inpututil"
	"github.com/hajimehoshi/ebiten/text"
	"github.com/lucasb-eyer/go-colorful"
	"golang.org/x/image/font/basicfont"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"time"
)

const (
	pixieMovementLength   = 5
	pixieStartingSize     = 4
	fairyDustSize         = 4
	maxFairyDustOnScreen  = 100
	fairyDustBatches      = 10
	fairyDustSpawnCadence = 5 * time.Second
)

type sprite struct {
	active bool

	x, y  int
	size  int
	color color.Color
}

type game struct {
	ticks int64
	level int

	lastFairyDustSpawnTime time.Time
	levelStartTime         time.Time
	levelDuration          time.Duration

	sx, sy int
	player *sprite

	fairyDust          [maxFairyDustOnScreen]*sprite
	fairyDustLeft      int
	fairyDustCollected int
	fairyDustAmount    int
}

func keyBeingPressed(key ebiten.Key) bool {
	return inpututil.KeyPressDuration(key) > 0
}

func intersects(p1, p2 sprite) bool {
	return ((p1.x >= p2.x && p1.x <= p2.x+p2.size) &&
		(p1.y >= p2.y && p1.y <= p2.y+p2.size)) ||
		((p2.x >= p1.x && p2.x <= p1.x+p1.size) &&
			(p2.y >= p1.y && p2.y <= p1.y+p1.size))
}

func (g game) newSprite(x, y int) *sprite {
	p := &sprite{}
	if x == -1 {
		p.x = rand.Intn(g.sx)
	} else {
		p.x = x
	}
	if y == -1 {
		p.y = rand.Intn(g.sy)
	} else {
		p.y = y
	}
	p.color = colorful.FastHappyColor()
	p.active = true
	return p
}

func (g game) newFairyDust(x, y int) *sprite {
	s := g.newSprite(x, y)
	s.size = fairyDustSize
	return s
}

func (g game) newPixie(x, y int) *sprite {
	s := g.newSprite(x, y)
	s.size = pixieStartingSize
	return s
}

func (p *sprite) grow() {
	p.size++
}

func (p *sprite) drawTo(i *ebiten.Image) {
	if !p.active {
		return
	}
	for x := 0; x < p.size; x++ {
		for y := 0; y < p.size; y++ {
			i.Set(p.x+x, p.y+y, p.color)
		}
	}
}

func (g *game) init(level int) {
	fairyDustCoefficient := level
	if level <= 0 {
		fairyDustCoefficient = 1
	} else if level > 10 {
		fairyDustCoefficient = 10
	}
	g.fairyDustAmount = fairyDustCoefficient * fairyDustBatches
	g.level = level
	g.sx, g.sy = ebiten.ScreenSizeInFullscreen()
	g.fairyDustLeft = g.fairyDustAmount
	g.player = g.newPixie(g.sx/20, g.sy/20)
	for i := 0; i < g.fairyDustAmount; i++ {
		g.fairyDust[i] = g.newFairyDust(-1, -1)
	}
	now := time.Now()
	g.lastFairyDustSpawnTime = now
	g.levelStartTime = now
}

func (g *game) Update(screen *ebiten.Image) (err error) {
	defer func() {
		if inpututil.IsKeyJustPressed(ebiten.KeyQ) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			os.Exit(0)
		} else if g.fairyDustLeft == 0 && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.level++
			g.init(g.level)
		}
		err = g.Draw(screen)
	}()
	if g.fairyDustLeft == 0 {
		return
	}

	g.ticks++
	if g.ticks == math.MaxInt64 {
		g.ticks = 0
	}
	now := time.Now()
	g.levelDuration = now.Sub(g.levelStartTime)

	playerPixie := g.player
	if keyBeingPressed(ebiten.KeyRight) && playerPixie.x+playerPixie.size <= g.sx {
		playerPixie.x = int(math.Min(float64(playerPixie.x+pixieMovementLength), float64(g.sx-playerPixie.size)))
	}
	if keyBeingPressed(ebiten.KeyLeft) && playerPixie.x >= 0 {
		playerPixie.x = int(math.Max(float64(playerPixie.x-pixieMovementLength), 0))
	}
	if keyBeingPressed(ebiten.KeyUp) && playerPixie.y >= 0 {
		playerPixie.y = int(math.Max(float64(playerPixie.y-pixieMovementLength), 0))
	}
	if keyBeingPressed(ebiten.KeyDown) && playerPixie.y+playerPixie.size <= g.sy {
		playerPixie.y = int(math.Min(float64(playerPixie.y+pixieMovementLength), float64(g.sy-playerPixie.size)))
	}

	for i := 0; i < g.fairyDustAmount; i++ {
		fd := g.fairyDust[i]
		if fd.active && intersects(*fd, *playerPixie) {
			if g.fairyDustCollected%4 == 0 {
				playerPixie.grow()
				playerPixie.color = fd.color
			}
			fd.active = false
			g.fairyDustLeft--
			g.fairyDustCollected++
		}
	}

	if g.ticks%100 == 0 {
		if now.Add(-fairyDustSpawnCadence).After(g.lastFairyDustSpawnTime) {
			for i := 0; i < g.fairyDustAmount; i++ {
				if !g.fairyDust[i].active {
					g.fairyDust[i] = g.newFairyDust(-1, -1)
					g.fairyDustLeft++
					g.lastFairyDustSpawnTime = now
					break
				}
			}
		}
	}
	return
}

func (g *game) Draw(screen *ebiten.Image) error {
	if err := screen.Fill(color.White); err != nil {
		return err
	}
	if g.fairyDustLeft != 0 {
		g.player.drawTo(screen)
		for i := 0; i < g.fairyDustAmount; i++ {
			g.fairyDust[i].drawTo(screen)
		}
	}
	text.Draw(screen, "pixie", basicfont.Face7x13, g.sx/2, 15, colorful.FastHappyColor())
	text.Draw(screen, fmt.Sprintf("fairy dust left: %d", g.fairyDustLeft), basicfont.Face7x13, 10, 15, colorful.FastHappyColor())
	text.Draw(screen, fmt.Sprintf("fairy dust collected: %d", g.fairyDustCollected), basicfont.Face7x13, 10, 30, colorful.FastHappyColor())
	hours := int(g.levelDuration.Hours())
	minutes := int(g.levelDuration.Minutes())
	secs := int(g.levelDuration.Seconds())
	text.Draw(screen, fmt.Sprintf("level %d", g.level), basicfont.Face7x13, g.sx-100, 15, colorful.FastHappyColor())
	text.Draw(screen, fmt.Sprintf("%02d:%02d:%02d", hours, minutes-(hours*60), secs-(minutes*60)), basicfont.Face7x13, g.sx-100, 30, colorful.FastHappyColor())
	if g.fairyDustLeft == 0 {
		text.Draw(screen, "complete!", basicfont.Face7x13, g.sx/2, g.sy/2, colorful.FastHappyColor())
	}
	return nil
}

func (g *game) Layout(_, _ int) (int, int) {
	return g.sx, g.sy
}

func main() {
	game := &game{}

	ebiten.SetWindowTitle("pixie")
	ebiten.SetFullscreen(true)
	ebiten.SetCursorMode(ebiten.CursorModeHidden)
	game.init(1)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatalln(err)
	}
}
