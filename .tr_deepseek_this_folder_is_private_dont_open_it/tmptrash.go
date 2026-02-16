

type ScreenshotOptions struct {

	OutputPath string

	Format proto.PageCaptureScreenshotFormat

	Quality int

	Clip *proto.PageViewport

	FullPage bool
}

func DefaultScreenshotOptions() ScreenshotOptions {
	return ScreenshotOptions{
		OutputPath: "screenshot.png",
		Format:     proto.PageCaptureScreenshotFormatPng,
		Quality:    90,
		FullPage:   true,
	}
}

func TakeScreenshot(url string, options ScreenshotOptions) (string, error) {
	if url == "" {
		return "", errors.New("URL cannot be empty")
	}

	if options.OutputPath == "" {
		options.OutputPath = "screenshot.png"
	}

	browser := rod.New().MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(url).MustWaitLoad()

	if options.Clip != nil || !options.FullPage {

		img, err := page.Screenshot(options.FullPage, &proto.PageCaptureScreenshot{
			Format:  options.Format,
			Quality: gson.Int(options.Quality),
			Clip:    options.Clip,
		})
		if err != nil {
			return "", err
		}

		if err := utils.OutputFile(options.OutputPath, img); err != nil {
			return "", err
		}
	} else {

		page.MustScreenshot(options.OutputPath)
	}

	if _, err := os.Stat(options.OutputPath); os.IsNotExist(err) {
		return "", errors.New("screenshot file was not created")
	}

	return options.OutputPath, nil
}

func TakeFullPageScreenshot(url, outputPath string) (string, error) {
	options := DefaultScreenshotOptions()
	options.OutputPath = outputPath
	return TakeScreenshot(url, options)
}

func TakePartialScreenshot(url, outputPath string, x, y, width, height float64) (string, error) {
	options := DefaultScreenshotOptions()
	options.OutputPath = outputPath
	options.FullPage = false
	options.Clip = &proto.PageViewport{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
		Scale:  1,
	}
	return TakeScreenshot(url, options)
}

