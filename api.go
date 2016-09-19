package main

func RunUtility(selfPath, utility string, args []string) error {
	manifestFinder, err := NewManifestFinder(selfPath)
	if err != nil {
		return err
	}

	nameVer := ParseName(utility)

	manifest, err := manifestFinder.Find(nameVer)
	if err != nil {
		return err
	}

	strategies, err := manifest.LoadStrategies(nameVer)
	if err != nil {
		return err
	}

	for _, strategy := range strategies {
		err = strategy.Run(args)
		if err != nil {
			// keep going if it's a reason to skip
			if _, ok := err.(*SkipError); !ok {
				return err
			}
		}
	}

	return nil
}
