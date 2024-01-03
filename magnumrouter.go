package magnumrouter

import "github.com/cassaram/quartz"

type MagnumRouter struct {
	address          string
	port             uint16
	conn             quartz.Quartz
	sourceNames      []string
	destinationNames []string
	destinationLocks []bool
	routeTable       [][]uint
	stop             bool
}

// Returns a reference to a new magnum router instance after configuration
// Level count is the number of levels supported by the quartz interface
// Typically 17 levels, 1 for video + 16 audio channels
// DestinationCount and SourceCount are the number of destinations / sources available in the Magnum interface
func NewMagnumRouter(address string, port uint16, sourceCount uint, destinationCount uint, levelCount uint) *MagnumRouter {
	r := MagnumRouter{
		address:          address,
		port:             port,
		conn:             *quartz.NewQuartz(address, port, true),
		sourceNames:      make([]string, sourceCount+1),
		destinationNames: make([]string, destinationCount+1),
		destinationLocks: make([]bool, destinationCount+1),
		routeTable:       make([][]uint, destinationCount+1),
		stop:             false,
	}
	for i := 0; i < len(r.routeTable); i++ {
		r.routeTable[i] = make([]uint, levelCount)
	}

	return &r
}

// Connect to the magnum server
// This will also try to pull all information from the server in terms of routes, names, and lock status
// Any errors will cause the connection to close and will be returned
func (m *MagnumRouter) Connect() error {
	m.stop = false
	go m.handleResponses(m.conn.RxMessages)
	err := m.conn.Connect()
	if err != nil {
		m.stop = true
		return err
	}

	// Get all inital information
	if m.RequestAllSourceNames() != nil {
		m.stop = true
		return err
	}
	if m.RequestAllDestinationNames() != nil {
		m.stop = true
		return err
	}
	if m.RequestAllDestinationLocks() != nil {
		m.stop = true
		return err
	}
	if m.RequestAllRoutes() != nil {
		m.stop = true
		return err
	}

	return nil
}

// Disconnect from the magnum server
func (m *MagnumRouter) Disconnect() error {
	m.stop = true
	return m.conn.Disconnect()
}

// Parses all return infromation from the server and stores it in cache
// Is automatically stopped / started with Connect() and Disconnect() methods
func (m *MagnumRouter) handleResponses(rxchan chan quartz.QuartzResponse) {
	for {
		if m.stop {
			return
		}

		msg := <-rxchan
		switch msg.GetType() {
		case quartz.QUARTZ_RESP_TYPE_ACK:
			// Ignore
		case quartz.QUARTZ_RESP_TYPE_ERR:
			// Ignore
		case quartz.QUARTZ_RESP_TYPE_PWRON:
			// Ignore
		case quartz.QUARTZ_RESP_TYPE_UPDATE:
			// Update our route table
			updateMsg := msg.(*quartz.ResponseUpdate)
			for _, level := range updateMsg.Levels {
				m.routeTable[updateMsg.Destination][quartzLevelToID(level)] = updateMsg.Source
			}
		case quartz.QUARTZ_RESP_TYPE_READ_DST:
			// Update name table
			nameMsg := msg.(*quartz.ResponseReadDestination)
			m.destinationNames[nameMsg.Destination] = nameMsg.Name
		case quartz.QUARTZ_RESP_TYPE_READ_SRC:
			// Update name table
			nameMsg := msg.(*quartz.ResponseReadSource)
			m.sourceNames[nameMsg.Source] = nameMsg.Name
		case quartz.QUARTZ_RESP_TYPE_READ_LVL:
			// Not supported by magnum, ignore
		case quartz.QUARTZ_RESP_TYPE_LOCK_STS:
			lockMsg := msg.(*quartz.ResponseLockStatus)
			m.destinationLocks[lockMsg.Destination] = lockMsg.Locked
		}
	}
}

// Request all source names from Magnum
// Results are cached and can be accessed via MagnumRouter.GetSourceNameTable() or MagnumRouter.GetSourceName(source)
func (m *MagnumRouter) RequestAllSourceNames() error {
	for i := 1; i < len(m.sourceNames); i++ {
		err := m.conn.GetSourceName(uint(i))
		if err != nil {
			return err
		}
	}
	return nil
}

// Request all destination names from Magnum
// Results are cached and can be accessed via MagnumRouter.GetDestinationNameTable() or MagnumRouter.GetDestinationName(destination)
func (m *MagnumRouter) RequestAllDestinationNames() error {
	for i := 1; i < len(m.destinationNames); i++ {
		err := m.conn.GetDestinationName(uint(i))
		if err != nil {
			return err
		}
	}
	return nil
}

// Request all destination locks from Magnum
// Results are cached and can be accessed via MagnumRouter.GetDestinationLockTable() or MagnumRouter.GetDestinationLock(destination)
func (m *MagnumRouter) RequestAllDestinationLocks() error {
	for i := 1; i < len(m.destinationLocks); i++ {
		err := m.conn.GetDestinationLock(uint(i))
		if err != nil {
			return err
		}
	}
	return nil
}

// Request all routes from Magnum
// Results are cached and can be accessed via MagnumRouter.GetRouteTable() or MagnumRouter.GetRoute(levels, destination)
func (m *MagnumRouter) RequestAllRoutes() error {
	for destIdx := 1; destIdx < len(m.routeTable); destIdx++ {
		for lvlIdx := 0; lvlIdx < len(m.routeTable[destIdx]); lvlIdx++ {
			err := m.conn.GetRoute(idToQuartzLevel(uint(lvlIdx)), uint(destIdx))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Returns the cached names of sources.
// Slice Index = Source ID
// Index 0 is unused due to it being reserved for magnum operations
func (m *MagnumRouter) GetSourceNameTable() []string {
	return m.sourceNames
}

// Returns the cached names of destinations.
// Slice Index = Destination ID
// Index 0 is unused due to it being reserved for magnum operations
func (m *MagnumRouter) GetDestinationNameTable() []string {
	return m.destinationNames
}

// Returns the cached destination lock status
// Slice Index = Destination ID
// Index 0 is unused due to it being reserved for magnum operations
func (m *MagnumRouter) GetDestinationLockTable() []bool {
	return m.destinationLocks
}

// Returns the cached route table.
// Dimension 0 indexes by destination ID
// Dimension 1 indexes by router level
// Dimension 1 values are source IDs
func (m *MagnumRouter) GetRouteTable() [][]uint {
	return m.routeTable
}

// Returns the cached source of a route
func (m *MagnumRouter) GetRoute(level uint, destination uint) uint {
	return m.routeTable[destination][level]
}

// Returns the cached name of a source
func (m *MagnumRouter) GetSourceName(source uint) string {
	return m.sourceNames[source]
}

// Returns the cached name of a destination
func (m *MagnumRouter) GetDestinationName(destination uint) string {
	return m.destinationNames[destination]
}

// Retruns whether a destination is locked or not
func (m *MagnumRouter) GetDestinationLocked(destination uint) bool {
	return m.destinationLocks[destination]
}

// Sets a crosspoint / route in magnum across defined level(s)
func (m *MagnumRouter) SetRoute(levels []uint, destination uint, source uint) error {
	quartzLevels := []quartz.QuartzLevel{}
	for _, lvl := range levels {
		quartzLevels = append(quartzLevels, idToQuartzLevel(lvl))
	}
	return m.conn.SetCrosspoint(quartzLevels, destination, source)
}

// Sets a lock status for a destination
func (m *MagnumRouter) SetLock(destination uint, lock bool) error {
	if lock {
		return m.conn.LockDestination(destination)
	} else {
		return m.conn.UnlockDestination(destination)
	}
}
