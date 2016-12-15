package server

type commChans struct {
	inChan  chan []byte
	outChan chan []byte
}

type Service func(*commChans)

func newCommChans() *commChans {
	return &commChans{
		inChan:  make(chan []byte),
		outChan: make(chan []byte),
	}
}

func (chans *commChans) Close() {
	close(chans.inChan)
	close(chans.outChan)
}
