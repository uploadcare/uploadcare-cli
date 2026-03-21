package cmd

import "github.com/spf13/cobra"

func newURLAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "url-api",
		Short: "URL API reference — on-the-fly CDN image transformations via URL construction",
		Long: `Uploadcare URL API performs on-the-fly image transformations by constructing
CDN URLs. No HTTP methods or request bodies — just URL path construction.

URL pattern:
  {cdn_base}/{uuid}/-/{operation}/{params}/-/{operation2}/{params2}/

Default CDN base: https://ucarecdn.com
Override with: --cdn-base flag, UPLOADCARE_CDN_BASE env var

Chain multiple operations by separating them with /-/ segments.
Operations are applied left to right. Trailing slash is required.

Operation categories:
  resize     resize, smart_resize, preview
  crop       crop, scale_crop
  rotate     rotate, flip, mirror, autorotate
  filter     blur, blur_region, sharp, grayscale, invert
  enhance    enhance
  color      brightness, contrast, saturation, warmth
  format     format, quality, progressive, strip_meta
  overlay    overlay
  other      border_radius, setfill, trim, rasterize

For the full machine-readable reference with all parameters and values:
  uploadcare api-schema | jq '.url_api'`,
		Example: `  # Resize to 800px wide, convert to WebP
  https://ucarecdn.com/{uuid}/-/resize/800x/-/format/webp/

  # Smart crop to 400x300, enhance, set quality
  https://ucarecdn.com/{uuid}/-/scale_crop/400x300/smart/-/enhance/-/quality/smart/

  # Grayscale with blur and rounded corners
  https://ucarecdn.com/{uuid}/-/grayscale/-/blur/50/-/border_radius/20/

  # Auto-rotate, resize, optimize for delivery
  https://ucarecdn.com/{uuid}/-/autorotate/yes/-/resize/1200x/-/quality/smart/-/format/auto/`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
}
