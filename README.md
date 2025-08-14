# skin64
Convert 64x32 Minecraft skin into 64x64 standard skin.

Support both slim and normal geometry models. Can fix the missing bottom skin part if it exists.

## Usage

```go
import (
	"image/png"
	"os"

	"github.com/redstonecraftgg/skin64"
)

func main() {
	f, _ := os.Open("skin.png")
	img, _ := png.Decode(f)
	out, _, _ := skin64.ConvertSize64(img)
	of, _ := os.Create("skin_out.png")
	png.Encode(of, out)
}
```
