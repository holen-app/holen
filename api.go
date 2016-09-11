package main

func RunUtility(utility string, args []string) error {
	manifestFinder, err := NewManifestFinder()
	if err != nil {
		return err
	}

	nameVer := ParseName(utility)

	manifest, err := manifestFinder.Find(nameVer)
	if err != nil {
		return err
	}

	strategy, err := manifest.LoadStrategy(nameVer)
	if err != nil {
		return err
	}

	return strategy.Run(args)
}
