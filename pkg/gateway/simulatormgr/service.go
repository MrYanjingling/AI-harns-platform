package simulatormgr

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/simonvetter/modbus"

	"harns-platform/pkg/gateway/simulatormgr/v1"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/utils/binutil"
)

type Service struct {
	mu        sync.RWMutex
	registers map[uint16]uint16
	points    map[uint16]Point
	server    *modbus.ModbusServer
	port      int
	running   bool
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

func NewService() *Service {
	return &Service{
		registers: make(map[uint16]uint16),
		points:    make(map[uint16]Point),
		port:      DefaultPort,
	}
}

func (s *Service) Open(ctx context.Context, port int) (*v1.StateVo, error) {
	if port == 0 {
		port = DefaultPort
	}
	if port < 1 || port > 65535 {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "port", MsgID: "validate.simulator_port_invalid",
		}})
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, validation.New([]validation.FieldIssue{{
			Field: "simulator", MsgID: "validate.simulator_already_running",
		}})
	}
	s.mu.Unlock()

	server, err := modbus.NewServer(&modbus.ServerConfiguration{
		URL:        fmt.Sprintf("tcp://0.0.0.0:%d", port),
		Timeout:    120 * time.Second,
		MaxClients: 10,
		Logger:     log.New(io.Discard, "", 0),
	}, s)
	if err != nil {
		return nil, err
	}
	if err := server.Start(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.server = server
	s.port = port
	s.running = true
	stopCh := make(chan struct{})
	s.stopCh = stopCh
	s.wg.Add(1)
	s.mu.Unlock()
	go s.simulateLoop(stopCh)
	return s.State(), nil
}

func (s *Service) Close(ctx context.Context) (*v1.StateVo, error) {
	s.mu.Lock()
	if !s.running {
		state := s.stateLocked()
		s.mu.Unlock()
		return state, nil
	}
	server := s.server
	stopCh := s.stopCh
	s.server = nil
	s.stopCh = nil
	s.running = false
	s.mu.Unlock()

	if server != nil {
		_ = server.Stop()
	}
	if stopCh != nil {
		close(stopCh)
	}
	s.wg.Wait()
	return s.State(), nil
}

func (s *Service) State() *v1.StateVo {
	if s == nil {
		return &v1.StateVo{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stateLocked()
}

func (s *Service) stateLocked() *v1.StateVo {
	addresses := make([]int, 0, len(s.points))
	for address := range s.points {
		addresses = append(addresses, int(address))
	}
	sort.Ints(addresses)
	points := make([]*v1.PointVo, 0, len(addresses))
	for _, raw := range addresses {
		address := uint16(raw)
		point := s.points[address]
		points = append(points, &v1.PointVo{
			Address:      address,
			Name:         point.Name,
			Description:  point.Description,
			RegisterType: point.RegisterType,
			DataType:     point.DataType,
			Mode:         string(point.Mode),
			Value:        s.pointValueLocked(address, point),
		})
	}
	return &v1.StateVo{
		Running:      s.running,
		Port:         s.port,
		UnitID:       DefaultUnitID,
		MemoryLayout: DefaultMemoryLayout,
		Points:       points,
	}
}

func (s *Service) AddPoint(address uint16, name, description, dataType string, mode Mode) (*v1.PointVo, error) {
	if address == 0 {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "address", MsgID: "validate.simulator_address_invalid",
		}})
	}
	dataType = strings.ToLower(strings.TrimSpace(dataType))
	if dataType == "" {
		dataType = DataTypeInt16
	} else if !validDataType(dataType) {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "dataType", MsgID: "validate.simulator_data_type_invalid",
		}})
	}
	if !validMode(mode) {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "mode", MsgID: "validate.simulator_mode_invalid",
		}})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	words := dataTypeWords(dataType)
	if uint32(address)+uint32(words) > 65536 {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "address", MsgID: "validate.simulator_address_invalid",
		}})
	}
	if s.rangeOccupiedLocked(address, words) {
		return nil, validation.New([]validation.FieldIssue{{
			Field: "address", MsgID: "validate.simulator_point_exists", Args: []interface{}{address},
		}})
	}
	s.points[address] = Point{
		Name:         strings.TrimSpace(name),
		Description:  description,
		RegisterType: RegisterTypeHolding,
		DataType:     dataType,
		Mode:         mode,
	}
	for index := uint16(0); index < words; index++ {
		if _, exists := s.registers[address+index]; !exists {
			s.registers[address+index] = 0
		}
	}
	point := s.points[address]
	return &v1.PointVo{
		Address:      address,
		Name:         point.Name,
		Description:  point.Description,
		RegisterType: point.RegisterType,
		DataType:     point.DataType,
		Mode:         string(point.Mode),
		Value:        s.pointValueLocked(address, point),
	}, nil
}

func (s *Service) rangeOccupiedLocked(address, words uint16) bool {
	start := uint32(address)
	end := start + uint32(words)
	for existingAddress, point := range s.points {
		existingStart := uint32(existingAddress)
		existingEnd := existingStart + uint32(dataTypeWords(point.DataType))
		if start < existingEnd && end > existingStart {
			return true
		}
	}
	return false
}

func (s *Service) pointValueLocked(address uint16, point Point) any {
	switch point.DataType {
	case DataTypeInt32:
		return s.readInt32Locked(address)
	case DataTypeFloat:
		return s.readFloatLocked(address)
	default:
		return s.registers[address]
	}
}

func (s *Service) registersBytesLocked(address, words uint16) []byte {
	data := make([]byte, words*2)
	for index := uint16(0); index < words; index++ {
		binutil.WriteUint16BigEndian(data[index*2:], s.registers[address+index])
	}
	return data
}

func (s *Service) readInt32Locked(address uint16) int32 {
	return int32(binutil.ParseUint32LittleEndianByteSwap(s.registersBytesLocked(address, 2)))
}

func (s *Service) writeInt32Locked(address uint16, value int32) {
	data := binutil.Uint32ToBytesLittleEndianByteSwap(uint32(value))
	s.registers[address] = binutil.ParseUint16BigEndian(data[:2])
	s.registers[address+1] = binutil.ParseUint16BigEndian(data[2:])
}

func (s *Service) readFloatLocked(address uint16) float32 {
	return binutil.ParseFloat32LittleEndianByteSwap(s.registersBytesLocked(address, 2))
}

func (s *Service) writeFloatLocked(address uint16, value float32) {
	data := binutil.Float32ToBytesLittleEndianByteSwap(value)
	s.registers[address] = binutil.ParseUint16BigEndian(data[:2])
	s.registers[address+1] = binutil.ParseUint16BigEndian(data[2:])
}
func (s *Service) DeletePoint(address uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.points[address]; !exists {
		return validation.New([]validation.FieldIssue{{
			Field: "address", MsgID: "validate.simulator_point_not_found", Args: []interface{}{address},
		}})
	}
	point := s.points[address]
	words := dataTypeWords(point.DataType)
	delete(s.points, address)
	for index := uint16(0); index < words; index++ {
		delete(s.registers, address+index)
	}
	return nil
}

func (s *Service) simulateLoop(stopCh <-chan struct{}) {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			s.simulateOnce()
		}
	}
}

func (s *Service) simulateOnce() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for address, point := range s.points {
		switch point.DataType {
		case DataTypeInt32:
			s.simulateInt32Locked(address, point.Mode)
		case DataTypeFloat:
			s.simulateFloatLocked(address, point.Mode)
		default:
			s.switchRegisterLocked(address, point.Mode)
		}
	}
}

func (s *Service) switchRegisterLocked(address uint16, mode Mode) {
	switch mode {
	case ModeRandom:
		s.registers[address] = uint16(rand.Uint32())
	case ModeIncrement:
		s.registers[address]++
	case ModeDecrement:
		s.registers[address]--
	}
}

func (s *Service) simulateFloatLocked(address uint16, mode Mode) {
	switch mode {
	case ModeRandom:
		s.writeFloatLocked(address, rand.Float32()*1000)
	case ModeIncrement:
		s.writeFloatLocked(address, s.readFloatLocked(address)+float32(0.1))
	case ModeDecrement:
		s.writeFloatLocked(address, s.readFloatLocked(address)-float32(0.1))
	}
}

func (s *Service) simulateInt32Locked(address uint16, mode Mode) {
	switch mode {
	case ModeRandom:
		s.writeInt32Locked(address, int32(rand.Uint32()))
	case ModeIncrement:
		s.writeInt32Locked(address, s.readInt32Locked(address)+1)
	case ModeDecrement:
		s.writeInt32Locked(address, s.readInt32Locked(address)-1)
	}
}

func (s *Service) HandleCoils(req *modbus.CoilsRequest) ([]bool, error) {
	return nil, modbus.ErrIllegalFunction
}

func (s *Service) HandleDiscreteInputs(req *modbus.DiscreteInputsRequest) ([]bool, error) {
	return nil, modbus.ErrIllegalFunction
}

func (s *Service) HandleHoldingRegisters(req *modbus.HoldingRegistersRequest) ([]uint16, error) {
	if req.UnitId != DefaultUnitID {
		return nil, modbus.ErrIllegalFunction
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.IsWrite {
		for index, value := range req.Args {
			s.registers[req.Addr+uint16(index)] = value
		}
		return nil, nil
	}
	result := make([]uint16, 0, req.Quantity)
	for index := uint16(0); index < req.Quantity; index++ {
		result = append(result, s.registers[req.Addr+index])
	}
	return result, nil
}

func (s *Service) HandleInputRegisters(req *modbus.InputRegistersRequest) ([]uint16, error) {
	if req.UnitId != DefaultUnitID {
		return nil, modbus.ErrIllegalFunction
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]uint16, 0, req.Quantity)
	for index := uint16(0); index < req.Quantity; index++ {
		result = append(result, s.registers[req.Addr+index])
	}
	return result, nil
}

var _ modbus.RequestHandler = (*Service)(nil)
