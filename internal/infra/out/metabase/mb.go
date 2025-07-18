package metabase

type Metabase struct{}

func New() *Metabase {
	return &Metabase{}
}

func Fetch() (any, error)

func FetchWithParametr() (any, error)
