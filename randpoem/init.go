package randpoem

import "github.com/bingoohuang/jj"

func init() {
	jj.RegisterSubstituteFn("唐诗", func(_ string) any { return RandPoetryTang() })
	jj.RegisterSubstituteFn("宋词", func(_ string) any { return RandSongci() })
	jj.RegisterSubstituteFn("诗经", func(_ string) any { return RandShijing() })
}
