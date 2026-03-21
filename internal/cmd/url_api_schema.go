package cmd

type urlAPISchema struct {
	Description string            `json:"description"`
	BaseURL     string            `json:"base_url"`
	URLPattern  string            `json:"url_pattern"`
	Notes       []string          `json:"notes"`
	Operations  []urlAPIOperation `json:"operations"`
	Examples    []urlAPIExample   `json:"examples"`
}

type urlAPIOperation struct {
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Format      string   `json:"format"`
	Values      []string `json:"values,omitempty"`
	Examples    []string `json:"examples"`
}

type urlAPIExample struct {
	Description string `json:"description"`
	URL         string `json:"url"`
}

func buildURLAPISchema() *urlAPISchema {
	return &urlAPISchema{
		Description: "Uploadcare URL API performs on-the-fly image transformations by constructing CDN URLs. No HTTP methods or request bodies — just URL path construction.",
		BaseURL:     "https://ucarecdn.com",
		URLPattern:  "{base_url}/{uuid}/-/{operation}/{params}/-/{operation2}/{params2}/",
		Notes: []string{
			"Chain multiple operations by separating them with /-/.",
			"Every operation segment starts with the operation name, followed by its parameters separated by /.",
			"The trailing slash is required.",
			"The base URL defaults to https://ucarecdn.com. Override with --cdn-base flag, UPLOADCARE_CDN_BASE env var, or it is auto-computed from your public key.",
			"The UUID is the file identifier returned by the Upload API or file list commands.",
			"Operations are applied in order from left to right.",
		},
		Operations: buildOperations(),
		Examples: []urlAPIExample{
			{
				Description: "Resize to 800px wide, convert to WebP",
				URL:         "https://ucarecdn.com/{uuid}/-/resize/800x/-/format/webp/",
			},
			{
				Description: "Smart crop to 400x300, enhance, set quality",
				URL:         "https://ucarecdn.com/{uuid}/-/scale_crop/400x300/smart/-/enhance/-/quality/smart/",
			},
			{
				Description: "Grayscale, blur, and add rounded corners",
				URL:         "https://ucarecdn.com/{uuid}/-/grayscale/-/blur/50/-/border_radius/20/",
			},
			{
				Description: "Overlay one image on another with offset and opacity",
				URL:         "https://ucarecdn.com/{uuid}/-/overlay/{overlay_uuid}/80p,90p/60x80/50p/",
			},
			{
				Description: "Full processing pipeline",
				URL:         "https://ucarecdn.com/{uuid}/-/autorotate/yes/-/resize/1200x/-/quality/smart/-/format/auto/",
			},
		},
	}
}

func buildOperations() []urlAPIOperation {
	return []urlAPIOperation{
		// Resize/crop
		{
			Name:        "resize",
			Category:    "resize",
			Description: "Resize the image to specified dimensions. Use WxH, Wx (width only), or xH (height only). Supports upscale flag.",
			Format:      "-/resize/{dimensions}/",
			Examples:    []string{"-/resize/800x600/", "-/resize/500x/", "-/resize/x400/"},
		},
		{
			Name:        "smart_resize",
			Category:    "resize",
			Description: "Content-aware resize that intelligently fills or removes areas to reach exact dimensions.",
			Format:      "-/smart_resize/{dimensions}/",
			Examples:    []string{"-/smart_resize/800x600/"},
		},
		{
			Name:        "crop",
			Category:    "crop",
			Description: "Crop the image to specified dimensions with optional offset. Offset can be in pixels or percentages.",
			Format:      "-/crop/{dimensions}/{offset}/",
			Examples:    []string{"-/crop/500x400/", "-/crop/500x400/100,50/", "-/crop/500x400/center/"},
		},
		{
			Name:        "scale_crop",
			Category:    "crop",
			Description: "Scale and crop to exact dimensions. Optional smart mode uses content detection for crop position.",
			Format:      "-/scale_crop/{dimensions}/{mode}/",
			Values:      []string{"center (default)", "smart"},
			Examples:    []string{"-/scale_crop/400x300/", "-/scale_crop/400x300/smart/"},
		},
		{
			Name:        "preview",
			Category:    "resize",
			Description: "Downscale the image proportionally to fit within the given dimensions.",
			Format:      "-/preview/{dimensions}/",
			Examples:    []string{"-/preview/800x600/"},
		},
		// Rotate/flip
		{
			Name:        "rotate",
			Category:    "rotate",
			Description: "Rotate the image by a multiple of 90 degrees.",
			Format:      "-/rotate/{angle}/",
			Values:      []string{"0", "90", "180", "270"},
			Examples:    []string{"-/rotate/90/", "-/rotate/270/"},
		},
		{
			Name:        "flip",
			Category:    "rotate",
			Description: "Flip the image vertically (top to bottom).",
			Format:      "-/flip/",
			Examples:    []string{"-/flip/"},
		},
		{
			Name:        "mirror",
			Category:    "rotate",
			Description: "Mirror the image horizontally (left to right).",
			Format:      "-/mirror/",
			Examples:    []string{"-/mirror/"},
		},
		{
			Name:        "autorotate",
			Category:    "rotate",
			Description: "Automatically rotate the image based on EXIF orientation data.",
			Format:      "-/autorotate/{enabled}/",
			Values:      []string{"yes", "no"},
			Examples:    []string{"-/autorotate/yes/"},
		},
		// Filter
		{
			Name:        "blur",
			Category:    "filter",
			Description: "Apply Gaussian blur. Strength from 0 to 5000.",
			Format:      "-/blur/{strength}/",
			Examples:    []string{"-/blur/50/", "-/blur/100/"},
		},
		{
			Name:        "blur_region",
			Category:    "filter",
			Description: "Apply Gaussian blur to a rectangular region. Dimensions and offset in pixels or percentages.",
			Format:      "-/blur_region/{dimensions}/{offset}/{strength}/",
			Examples:    []string{"-/blur_region/200x100/30,30/50/", "-/blur_region/50px50p/25p,25p/80/"},
		},
		{
			Name:        "sharp",
			Category:    "filter",
			Description: "Sharpen the image. Strength from 0 to 20.",
			Format:      "-/sharp/{strength}/",
			Examples:    []string{"-/sharp/5/", "-/sharp/10/"},
		},
		{
			Name:        "grayscale",
			Category:    "filter",
			Description: "Convert the image to grayscale.",
			Format:      "-/grayscale/",
			Examples:    []string{"-/grayscale/"},
		},
		{
			Name:        "invert",
			Category:    "filter",
			Description: "Invert all colors in the image.",
			Format:      "-/invert/",
			Examples:    []string{"-/invert/"},
		},
		// Enhance/color
		{
			Name:        "enhance",
			Category:    "enhance",
			Description: "Auto-enhance the image (auto levels, color correction).",
			Format:      "-/enhance/{strength}/",
			Examples:    []string{"-/enhance/", "-/enhance/50/"},
		},
		{
			Name:        "brightness",
			Category:    "color",
			Description: "Adjust brightness. Range: -100 to 100.",
			Format:      "-/brightness/{value}/",
			Examples:    []string{"-/brightness/30/", "-/brightness/-20/"},
		},
		{
			Name:        "contrast",
			Category:    "color",
			Description: "Adjust contrast. Range: -100 to 100.",
			Format:      "-/contrast/{value}/",
			Examples:    []string{"-/contrast/20/", "-/contrast/-10/"},
		},
		{
			Name:        "saturation",
			Category:    "color",
			Description: "Adjust color saturation. Range: -100 to 100.",
			Format:      "-/saturation/{value}/",
			Examples:    []string{"-/saturation/30/", "-/saturation/-50/"},
		},
		{
			Name:        "warmth",
			Category:    "color",
			Description: "Adjust color temperature (warmth). Range: -100 to 100.",
			Format:      "-/warmth/{value}/",
			Examples:    []string{"-/warmth/20/", "-/warmth/-30/"},
		},
		// Format
		{
			Name:        "format",
			Category:    "format",
			Description: "Convert the image to the specified format. 'auto' selects the best format based on the client.",
			Format:      "-/format/{format}/",
			Values:      []string{"jpeg", "png", "webp", "avif", "auto"},
			Examples:    []string{"-/format/webp/", "-/format/auto/"},
		},
		{
			Name:        "quality",
			Category:    "format",
			Description: "Set output quality for lossy formats. 'smart' auto-selects optimal quality.",
			Format:      "-/quality/{value}/",
			Values:      []string{"smart", "smart_retina", "normal", "lighter", "lightest"},
			Examples:    []string{"-/quality/smart/", "-/quality/lighter/"},
		},
		{
			Name:        "progressive",
			Category:    "format",
			Description: "Enable progressive encoding for JPEG output.",
			Format:      "-/progressive/{enabled}/",
			Values:      []string{"yes", "no"},
			Examples:    []string{"-/progressive/yes/"},
		},
		{
			Name:        "strip_meta",
			Category:    "format",
			Description: "Strip EXIF and other metadata from the output image.",
			Format:      "-/strip_meta/{enabled}/",
			Values:      []string{"true", "false"},
			Examples:    []string{"-/strip_meta/true/"},
		},
		// Overlay
		{
			Name:        "overlay",
			Category:    "overlay",
			Description: "Overlay another image on top. Specify overlay UUID, optional dimensions, offset, and opacity (0-100).",
			Format:      "-/overlay/{uuid}/{dimensions}/{offset}/{opacity}p/",
			Examples:    []string{"-/overlay/{overlay_uuid}/50px50p/10,10/80p/"},
		},
		// Other
		{
			Name:        "border_radius",
			Category:    "other",
			Description: "Round the corners of the image. Value in pixels or percentage.",
			Format:      "-/border_radius/{radii}/",
			Examples:    []string{"-/border_radius/20/", "-/border_radius/50p/"},
		},
		{
			Name:        "setfill",
			Category:    "other",
			Description: "Set fill color for transparent areas or padding. Hex color without #.",
			Format:      "-/setfill/{color}/",
			Examples:    []string{"-/setfill/ffffff/", "-/setfill/ff0000/"},
		},
		{
			Name:        "trim",
			Category:    "other",
			Description: "Remove uniform-colored borders from the image.",
			Format:      "-/trim/{mode}/",
			Values:      []string{"auto"},
			Examples:    []string{"-/trim/auto/"},
		},
		{
			Name:        "rasterize",
			Category:    "other",
			Description: "Rasterize SVG or PDF files to a raster image format for further processing.",
			Format:      "-/rasterize/",
			Examples:    []string{"-/rasterize/"},
		},
	}
}
