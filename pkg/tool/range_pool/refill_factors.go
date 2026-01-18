package range_pool

func computeRefillFactors(windowLen int64, tiers []TierConfig) (int64, int64) {
	topSize, topMax := topTierConfig(tiers)
	if topSize <= 0 {
		return 2, 4
	}
	if topMax <= 0 {
		topMax = 1
	}

	// Aim to keep at least ~1.2 * (topSize * topMax) remaining before triggering.
	triggerLen := topSize * topMax * 12 / 10
	if windowLen > 0 && triggerLen > windowLen/2 {
		triggerLen = windowLen / 2
	}
	triggerFactor := triggerLen / topSize
	if triggerFactor < 2 {
		triggerFactor = 2
	}

	// Refill should exceed trigger threshold by at least ~0.5 * (topSize * topMax).
	refillLen := triggerLen + (topSize*topMax)/2
	refillFactor := refillLen / topSize
	if refillFactor < 4 {
		refillFactor = 4
	}
	if refillFactor <= triggerFactor {
		refillFactor = triggerFactor + 2
	}
	return triggerFactor, refillFactor
}

func topTierConfig(tiers []TierConfig) (int64, int64) {
	var topSize int64
	var topMax int64
	for _, tier := range tiers {
		if tier.Size > topSize {
			topSize = tier.Size
			maxCount := tier.MaxCount
			if maxCount <= 0 {
				maxCount = tier.Threshold + 1
			}
			topMax = int64(maxCount)
		}
	}
	return topSize, topMax
}
