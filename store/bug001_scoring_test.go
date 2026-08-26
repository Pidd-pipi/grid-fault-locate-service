package store

import (
	"sync"
	"testing"
	"time"

	"example.com/grid-fault-locate-service/domain"
)

func bug001NewStore(t *testing.T) *Store {
	t.Helper()
	st, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func bug001Fault(id, feederID, sectionID string) *domain.FaultEvent {
	return domain.NewFaultEvent(id, feederID, "测试线", sectionID, []string{sectionID}, nil, nil, "ev", "op", time.Now())
}

func TestBug001ListActiveFaultsRace(t *testing.T) {
	st := bug001NewStore(t)
	_ = st.CreateFault(bug001Fault("FE-1", "F1", "S1"))
	start := make(chan struct{})
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for {
			select {
			case <-stop:
				return
			default:
				_ = st.ListActiveFaults()
			}
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				i++
				_ = st.CreateFault(bug001Fault("FE-"+string(rune('A'+i%20)), "F1", "S1"))
			}
		}
	}()
	close(start)
	time.Sleep(40 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestBug001ListFaultsRace(t *testing.T) {
	st := bug001NewStore(t)
	_ = st.CreateFault(bug001Fault("FE-1", "F1", "S1"))
	start := make(chan struct{})
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for {
			select {
			case <-stop:
				return
			default:
				_ = st.ListFaults("", "")
			}
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				i++
				_ = st.CreateFault(bug001Fault("FE-"+string(rune('A'+i%20)), "F1", "S1"))
			}
		}
	}()
	close(start)
	time.Sleep(40 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestBug001ListIndicatorsRace(t *testing.T) {
	st := bug001NewStore(t)
	_ = st.CreateIndicator(&domain.FaultIndicator{ID: "FI-1", FeederID: "F1", SectionID: "S1", Name: "i1", Status: domain.IndicatorReset})
	start := make(chan struct{})
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for {
			select {
			case <-stop:
				return
			default:
				_ = st.ListIndicators("", "")
			}
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				i++
				_ = st.CreateIndicator(&domain.FaultIndicator{ID: "FI-" + string(rune('A'+i%20)), FeederID: "F1", SectionID: "S1", Name: "i", Status: domain.IndicatorReset})
			}
		}
	}()
	close(start)
	time.Sleep(40 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestBug001CountTriggeredIndicatorsRace(t *testing.T) {
	st := bug001NewStore(t)
	_ = st.CreateIndicator(&domain.FaultIndicator{ID: "FI-1", FeederID: "F1", SectionID: "S1", Name: "i1", Status: domain.IndicatorTriggered})
	start := make(chan struct{})
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for {
			select {
			case <-stop:
				return
			default:
				_ = st.CountTriggeredIndicators()
			}
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				i++
				_ = st.CreateIndicator(&domain.FaultIndicator{ID: "FI-" + string(rune('A'+i%20)), FeederID: "F1", SectionID: "S1", Name: "i", Status: domain.IndicatorReset})
			}
		}
	}()
	close(start)
	time.Sleep(40 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestBug001ListFeedersRace(t *testing.T) {
	st := bug001NewStore(t)
	_ = st.CreateFeeder(&domain.Feeder{ID: "F1", Name: "n", Substation: "s", VoltageLevel: "10kV", Status: domain.FeederActive})
	start := make(chan struct{})
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for {
			select {
			case <-stop:
				return
			default:
				_ = st.ListFeeders()
			}
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				i++
				_ = st.CreateFeeder(&domain.Feeder{ID: "F" + string(rune('A'+i%20)), Name: "n", Substation: "s", VoltageLevel: "10kV", Status: domain.FeederActive})
			}
		}
	}()
	close(start)
	time.Sleep(40 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestBug001GetFeederRace(t *testing.T) {
	st := bug001NewStore(t)
	_ = st.CreateFeeder(&domain.Feeder{ID: "F1", Name: "n", Substation: "s", VoltageLevel: "10kV", Status: domain.FeederActive})
	start := make(chan struct{})
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for {
			select {
			case <-stop:
				return
			default:
				_, _ = st.GetFeeder("F1")
			}
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				i++
				_ = st.CreateFeeder(&domain.Feeder{ID: "F" + string(rune('A'+i%20)), Name: "n", Substation: "s", VoltageLevel: "10kV", Status: domain.FeederActive})
			}
		}
	}()
	close(start)
	time.Sleep(40 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestBug001ListSwitchesRace(t *testing.T) {
	st := bug001NewStore(t)
	_ = st.CreateSwitch(&domain.SwitchNode{ID: "SW-1", FeederID: "F1", Name: "sw", SwitchType: domain.SwitchTypeSectionalizer, Status: domain.SwitchClosed})
	start := make(chan struct{})
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for {
			select {
			case <-stop:
				return
			default:
				_ = st.ListSwitches("")
			}
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				i++
				_ = st.CreateSwitch(&domain.SwitchNode{ID: "SW-" + string(rune('A'+i%20)), FeederID: "F1", Name: "sw", SwitchType: domain.SwitchTypeSectionalizer, Status: domain.SwitchClosed})
			}
		}
	}()
	close(start)
	time.Sleep(40 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestBug001CountSwitchesOfFeederRace(t *testing.T) {
	st := bug001NewStore(t)
	_ = st.CreateSwitch(&domain.SwitchNode{ID: "SW-1", FeederID: "F1", Name: "sw", SwitchType: domain.SwitchTypeSectionalizer, Status: domain.SwitchClosed})
	start := make(chan struct{})
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for {
			select {
			case <-stop:
				return
			default:
				_ = st.CountSwitchesOfFeeder("F1")
			}
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				i++
				_ = st.CreateSwitch(&domain.SwitchNode{ID: "SW-" + string(rune('A'+i%20)), FeederID: "F1", Name: "sw", SwitchType: domain.SwitchTypeSectionalizer, Status: domain.SwitchClosed})
			}
		}
	}()
	close(start)
	time.Sleep(40 * time.Millisecond)
	close(stop)
	wg.Wait()
}
