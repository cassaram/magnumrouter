package magnumrouter

import (
	"strings"

	"github.com/cassaram/quartz"
)

func quartzLevelToID(level quartz.QuartzLevel) uint {
	levelIds := "VABCDEFGHIJKLMNOPQRSTUWXYZ"
	return uint(strings.Index(levelIds, string(level)))
}

func idToQuartzLevel(id uint) quartz.QuartzLevel {
	levelIds := "VABCDEFGHIJKLMNOPQRSTUWXYZ"
	return quartz.QuartzLevel(levelIds[id])
}
