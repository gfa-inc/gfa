package gfa

import (
	"github.com/common-nighthawk/go-figure"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
)

type bannerConfig struct {
	Text    string
	Enabled bool
}

func printBanner() {
	option := bannerConfig{
		Enabled: true,
		Text:    "GFA",
	}
	err := config.UnmarshalKey("banner", &option)
	if err != nil {
		logger.Debug(err)
		return
	}

	if !option.Enabled {
		logger.Debug("Banner is disabled")
		return
	}

	name := config.GetString("name")
	if name != "" {
		option.Text = name
	}
	figure.NewFigure(option.Text, "", true).Print()
}
