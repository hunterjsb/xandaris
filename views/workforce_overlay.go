package views

import (
	"fmt"
	"image"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hunterjsb/xandaris/entities"
	"github.com/hunterjsb/xandaris/utils"
)

type workforceRow struct {
	Building *entities.Building
	Assigned int
	Required int
	Desired  int
}

type workforceRowHit struct {
	Minus   image.Rectangle
	Plus    image.Rectangle
	Auto    image.Rectangle
	Disable image.Rectangle
}

type WorkforceOverlay struct {
	planet *entities.Planet

	show   bool
	hits   map[*entities.Building]workforceRowHit
	rect   image.Rectangle
	margin int
}

func NewWorkforceOverlay() *WorkforceOverlay {
	margin := 60
	return &WorkforceOverlay{
		show:   false,
		hits:   make(map[*entities.Building]workforceRowHit),
		margin: margin,
		rect:   image.Rect(margin, margin, ScreenWidth-margin, ScreenHeight-margin),
	}
}

func (wo *WorkforceOverlay) SetPlanet(planet *entities.Planet) {
	wo.planet = planet
	wo.hits = make(map[*entities.Building]workforceRowHit)
}

func (wo *WorkforceOverlay) Toggle() {
	wo.show = !wo.show
	if !wo.show {
		wo.hits = make(map[*entities.Building]workforceRowHit)
	}
}

func (wo *WorkforceOverlay) Hide() {
	wo.show = false
	wo.hits = make(map[*entities.Building]workforceRowHit)
}

func (wo *WorkforceOverlay) Visible() bool {
	return wo.show
}

func (wo *WorkforceOverlay) Rect() image.Rectangle {
	return wo.rect
}

func (wo *WorkforceOverlay) Draw(screen *ebiten.Image) {
	if !wo.show || wo.planet == nil {
		return
	}

	wo.planet.RebalanceWorkforce()
	wo.hits = make(map[*entities.Building]workforceRowHit)

	panel := NewUIPanel(wo.rect.Min.X, wo.rect.Min.Y, wo.rect.Dx(), wo.rect.Dy())
	panel.BgColor = color.RGBA{18, 20, 32, 235}
	panel.Draw(screen)

	DrawText(screen, "Population & Workforce Overview", wo.rect.Min.X+30, wo.rect.Min.Y+40, utils.TextPrimary)

	contentX := wo.rect.Min.X + 30
	contentWidth := wo.rect.Dx() - 60
	leftWidth := int(float64(contentWidth) * 0.55)
	if leftWidth < 260 {
		leftWidth = 260
	}
	gap := 40
	rightX := contentX + leftWidth + gap
	rightWidth := wo.rect.Max.X - rightX - 30
	if rightWidth < 220 {
		rightWidth = 220
		leftWidth = contentWidth - rightWidth - gap
	}

	leftY := wo.rect.Min.Y + 90
	rightY := leftY

	capacity := wo.planet.GetTotalPopulationCapacity()
	DrawText(screen, "Population", contentX, leftY, utils.TextSecondary)
	leftY += 20
	popBar := NewUIProgressBar(contentX, leftY, leftWidth, 18)
	maxPop := float64(capacity)
	if maxPop < 1 {
		maxPop = 1
	}
	popBar.SetValue(float64(wo.planet.Population), maxPop)
	popBar.FillColor = utils.PlayerGreen
	popBar.Draw(screen)
	DrawText(
		screen,
		fmt.Sprintf("%s / %s", utils.FormatInt64WithCommas(wo.planet.Population), utils.FormatInt64WithCommas(capacity)),
		contentX,
		leftY+26,
		utils.TextSecondary,
	)
	leftY += 58

	housingSegments := buildHousingSegments(wo.planet)
	if len(housingSegments) > 0 {
		DrawText(screen, "Housing Sources", contentX, leftY, utils.TextSecondary)
		leftY += 20
		DrawStackedBar(screen, image.Rect(contentX, leftY, contentX+leftWidth, leftY+18), housingSegments, color.RGBA{22, 22, 38, 255}, utils.PanelBorder)
		legendBottom := DrawLegend(screen, image.Point{X: contentX, Y: leftY + 24}, housingSegments)
		leftY = legendBottom + 28
	}

	DrawText(screen, "Workforce", contentX, leftY, utils.TextSecondary)
	leftY += 20
	workforceBar := NewUIProgressBar(contentX, leftY, leftWidth, 18)
	maxWorkers := float64(wo.planet.WorkforceTotal)
	if maxWorkers < 1 {
		maxWorkers = 1
	}
	workforceBar.SetValue(float64(wo.planet.WorkforceUsed), maxWorkers)
	workforceBar.FillColor = utils.PlayerBlue
	workforceBar.Draw(screen)
	DrawText(
		screen,
		fmt.Sprintf("%s used / %s total  (Available: %s)",
			utils.FormatInt64WithCommas(wo.planet.WorkforceUsed),
			utils.FormatInt64WithCommas(wo.planet.WorkforceTotal),
			utils.FormatInt64WithCommas(wo.planet.GetAvailableWorkforce())),
		contentX,
		leftY+26,
		utils.TextSecondary,
	)

	DrawText(screen, "Building Staffing", rightX, rightY, utils.TextSecondary)
	rightY += 28

	rows := buildWorkforceRows(wo.planet)
	if len(rows) == 0 {
		DrawText(screen, "No staffed buildings", rightX, rightY, utils.TextSecondary)
	} else {
		buttonWidth := 42
		buttonHeight := 22
		controlAreaWidth := buttonWidth*4 + 18
		progressWidth := rightWidth - controlAreaWidth - 16
		if progressWidth < 120 {
			progressWidth = 120
		}

		for _, row := range rows {
			building := row.Building
			DrawText(screen, fmt.Sprintf("%s (%s)", building.Name, building.BuildingType), rightX, rightY, utils.TextPrimary)
			rightY += 18

			bar := NewUIProgressBar(rightX, rightY, progressWidth, 16)
			bar.SetValue(float64(row.Assigned), float64(maxInt(row.Required, 1)))
			bar.FillColor = colorForWorkforceRatio(row.Assigned, row.Required)
			bar.Draw(screen)

			DrawText(screen,
				fmt.Sprintf("%s / %s assigned", utils.FormatIntWithCommas(row.Assigned), utils.FormatIntWithCommas(row.Required)),
				rightX,
				rightY+22,
				utils.TextSecondary,
			)

			buttonY := rightY
			buttonX := rightX + progressWidth + 12
			minusRect := image.Rect(buttonX, buttonY, buttonX+buttonWidth, buttonY+buttonHeight)
			buttonX += buttonWidth + 4
			plusRect := image.Rect(buttonX, buttonY, buttonX+buttonWidth, buttonY+buttonHeight)
			buttonX += buttonWidth + 4
			autoRect := image.Rect(buttonX, buttonY, buttonX+buttonWidth, buttonY+buttonHeight)
			buttonX += buttonWidth + 4
			disableRect := image.Rect(buttonX, buttonY, buttonX+buttonWidth, buttonY+buttonHeight)

			DrawLabeledButton(screen, minusRect, "-", false)
			DrawLabeledButton(screen, plusRect, "+", false)
			DrawLabeledButton(screen, autoRect, "Auto", row.Desired < 0)
			DrawLabeledButton(screen, disableRect, "Off", row.Desired == 0)

			wo.hits[building] = workforceRowHit{
				Minus:   minusRect,
				Plus:    plusRect,
				Auto:    autoRect,
				Disable: disableRect,
			}

			desiredLabel := "Desired: Auto"
			if row.Desired == 0 {
				desiredLabel = "Desired: Off"
			} else if row.Desired > 0 {
				desiredLabel = fmt.Sprintf("Desired: %s", utils.FormatIntWithCommas(row.Desired))
			}
			DrawText(screen, desiredLabel, rightX, rightY+40, utils.TextSecondary)

			rightY += 52
		}
	}

	DrawText(screen, "Press W or ESC to close", wo.rect.Min.X+30, wo.rect.Max.Y-30, utils.TextSecondary)
}

func (wo *WorkforceOverlay) HandleClick(mx, my int) bool {
	if !wo.show || wo.planet == nil {
		return false
	}
	if !rectContains(mx, my, wo.rect) {
		return false
	}
	if wo.hits == nil {
		return true
	}

	for building, hits := range wo.hits {
		switch {
		case rectContains(mx, my, hits.Auto):
			building.SetDesiredWorkers(-1)
		case rectContains(mx, my, hits.Disable):
			building.SetDesiredWorkers(0)
		case rectContains(mx, my, hits.Minus):
			step := workforceAdjustStep(building)
			desired := building.DesiredWorkers
			if desired < 0 {
				desired = building.WorkersRequired
			}
			desired -= step
			if desired < 0 {
				desired = 0
			}
			building.SetDesiredWorkers(desired)
		case rectContains(mx, my, hits.Plus):
			step := workforceAdjustStep(building)
			desired := building.DesiredWorkers
			if desired < 0 {
				desired = building.WorkersRequired
			}
			desired += step
			if desired > building.WorkersRequired {
				desired = building.WorkersRequired
			}
			building.SetDesiredWorkers(desired)
		default:
			continue
		}
		wo.planet.RebalanceWorkforce()
		return true
	}

	return true
}

// helper functions (reused from planet_view, now local to overlay)

func buildHousingSegments(planet *entities.Planet) []ChartSegment {
	segments := make([]ChartSegment, 0)

	baseCap := planet.GetBaseHousingCapacity()
	if baseCap > 0 {
		segments = append(segments, ChartSegment{
			Label: "Base",
			Value: float64(baseCap),
			Color: colorForBuildingType("Base"),
		})
	}

	typeSums := make(map[string]float64)
	for _, entity := range planet.Buildings {
		building, ok := entity.(*entities.Building)
		if !ok {
			continue
		}
		if building.BuildingType == "Base" {
			continue
		}
		if building.PopulationCapacity <= 0 {
			continue
		}
		cap := float64(building.GetEffectivePopulationCapacity())
		if cap <= 0 {
			continue
		}
		typeSums[building.BuildingType] += cap
	}

	labels := make([]string, 0, len(typeSums))
	for label := range typeSums {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	for _, label := range labels {
		segments = append(segments, ChartSegment{
			Label: label,
			Value: typeSums[label],
			Color: colorForBuildingType(label),
		})
	}

	return segments
}

func buildWorkforceRows(planet *entities.Planet) []workforceRow {
	rows := make([]workforceRow, 0)

	for _, entity := range planet.Buildings {
		building, ok := entity.(*entities.Building)
		if !ok {
			continue
		}
		if building.WorkersRequired <= 0 {
			continue
		}
		rows = append(rows, workforceRow{
			Building: building,
			Assigned: building.WorkersAssigned,
			Required: building.WorkersRequired,
			Desired:  building.DesiredWorkers,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		bi := rows[i].Building
		bj := rows[j].Building
		if bi.BuildingType == bj.BuildingType {
			return bi.Name < bj.Name
		}
		if bi.BuildingType == "Base" {
			return true
		}
		if bj.BuildingType == "Base" {
			return false
		}
		return bi.BuildingType < bj.BuildingType
	})

	return rows
}

func colorForBuildingType(buildingType string) color.RGBA {
	switch buildingType {
	case "Base":
		return utils.PlayerBlue
	case "Habitat":
		return utils.PlayerGreen
	case "Mine":
		return utils.StationMining
	case "Refinery":
		return utils.StationRefinery
	case "Shipyard":
		return utils.StationShipyard
	case "Trading Post":
		return utils.StationTrading
	default:
		return utils.Highlight
	}
}

func colorForWorkforceRatio(assigned, required int) color.RGBA {
	if required <= 0 {
		return utils.PlayerGreen
	}
	ratio := float64(assigned) / float64(required)
	if ratio >= 0.95 {
		return utils.PlayerGreen
	}
	if ratio >= 0.5 {
		return utils.StationRefinery
	}
	return color.RGBA{200, 80, 80, 255}
}

func workforceAdjustStep(building *entities.Building) int {
	if building == nil {
		return 1
	}
	step := building.WorkersRequired / 10
	if step < 5 {
		step = 5
	}
	if step > building.WorkersRequired {
		step = building.WorkersRequired
	}
	if step < 1 {
		step = 1
	}
	return step
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
