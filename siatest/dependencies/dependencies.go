package dependencies

import (
	"net"
	"sync"

	"gitlab.com/NebulousLabs/Sia/modules"
)

type (
	// DependencyBlockScan blocks the scan progress of the hostdb until Scan is
	// called on the dependency.
	DependencyBlockScan struct {
		modules.ProductionDependencies
		closed bool
		c      chan struct{}
	}

	// DependencyDisableCloseUploadEntry prevents SiaFileEntries in the upload code
	// from being closed.
	DependencyDisableCloseUploadEntry struct {
		modules.ProductionDependencies
	}

	// DependencyDisableContractRecovery prevents recoverable contracts from being
	// recovered in threadedContractMaintenance.
	DependencyDisableContractRecovery struct {
		modules.ProductionDependencies
	}

	// DependencyDisableRecoveryStatusReset prevents the fields scanInProgress and
	// atomicRecoveryScanHeight from being reset after the scan is done.
	DependencyDisableRecoveryStatusReset struct {
		modules.ProductionDependencies
	}

	// DependencyDisableRenewal prevents contracts from being renewed.
	DependencyDisableRenewal struct {
		modules.ProductionDependencies
	}

	// DependencyDisableRepairAndHealthLoops prevents the background loops for
	// repairs and updating directory metadata from running. This includes
	// threadedUploadAndRepair, threadedStuckLoop, and
	// threadedUpdateRenterHealth
	DependencyDisableRepairAndHealthLoops struct {
		modules.ProductionDependencies
	}

	// DependencyInterruptOnceOnKeyword is a generic dependency that interrupts
	// the flow of the program if the argument passed to Disrupt equals str and
	// if f was set to true by calling Fail.
	DependencyInterruptOnceOnKeyword struct {
		f bool // indicates if the next download should fail
		modules.ProductionDependencies
		mu  sync.Mutex
		str string
	}

	// DependencyInterruptAfterNCalls is a generic dependency that behaves the same
	// way as DependencyInterruptOnceOnKeyword, expect that after calling "Fail",
	// "Disrupt" needs to be called n times for the actual disrupt to happen.
	DependencyInterruptAfterNCalls struct {
		DependencyInterruptOnceOnKeyword
		n    int
		cntr int
	}
)

// NewDependencyCustomResolver creates a dependency from a given lookupIP
// method which returns a custom resolver that uses the specified lookupIP
// method to resolve hostnames.
func NewDependencyCustomResolver(lookupIP func(string) ([]net.IP, error)) modules.Dependencies {
	return &dependencyCustomResolver{lookupIP: lookupIP}
}

// NewDependencyDisruptUploadStream creates a new dependency that closes the
// reader used for upload streaming to simulate failing connection after
// numChunks uploaded chunks.
func NewDependencyDisruptUploadStream(numChunks int) *DependencyInterruptAfterNCalls {
	return newDependencyInterruptAfterNCalls("DisruptUploadStream", numChunks)
}

// NewDependencyInterruptContractSaveToDiskAfterDeletion creates a new
// dependency that interrupts the contract being saved to disk after being
// removed from static contracts
func NewDependencyInterruptContractSaveToDiskAfterDeletion() *DependencyInterruptOnceOnKeyword {
	return newDependencyInterruptOnceOnKeyword("InterruptContractSaveToDiskAfterDeletion")
}

// NewDependencyInterruptDownloadBeforeSendingRevision creates a new dependency
// that interrupts the download on the renter side before sending the signed
// revision to the host.
func NewDependencyInterruptDownloadBeforeSendingRevision() *DependencyInterruptOnceOnKeyword {
	return newDependencyInterruptOnceOnKeyword("InterruptDownloadBeforeSendingRevision")
}

// NewDependencyInterruptDownloadAfterSendingRevision creates a new dependency
// that interrupts the download on the renter side right after receiving the
// signed revision from the host.
func NewDependencyInterruptDownloadAfterSendingRevision() *DependencyInterruptOnceOnKeyword {
	return newDependencyInterruptOnceOnKeyword("InterruptDownloadAfterSendingRevision")
}

// NewDependencyInterruptUploadBeforeSendingRevision creates a new dependency
// that interrupts the upload on the renter side before sending the signed
// revision to the host.
func NewDependencyInterruptUploadBeforeSendingRevision() *DependencyInterruptOnceOnKeyword {
	return newDependencyInterruptOnceOnKeyword("InterruptUploadBeforeSendingRevision")
}

// NewDependencyInterruptUploadAfterSendingRevision creates a new dependency
// that interrupts the upload on the renter side right after receiving the
// signed revision from the host.
func NewDependencyInterruptUploadAfterSendingRevision() *DependencyInterruptOnceOnKeyword {
	return newDependencyInterruptOnceOnKeyword("InterruptUploadAfterSendingRevision")
}

// newDependencyInterruptOnceOnKeyword creates a new
// DependencyInterruptOnceOnKeyword from a given disrupt key.
func newDependencyInterruptOnceOnKeyword(str string) *DependencyInterruptOnceOnKeyword {
	return &DependencyInterruptOnceOnKeyword{
		str: str,
	}
}

// newDependencyInterruptAfterNCalls creates a new
// DependencyInterruptAfterNCalls from a given disrupt key and n.
func newDependencyInterruptAfterNCalls(str string, n int) *DependencyInterruptAfterNCalls {
	return &DependencyInterruptAfterNCalls{
		DependencyInterruptOnceOnKeyword: DependencyInterruptOnceOnKeyword{
			str: str,
		},
		n: n,
	}
}

// Disrupt returns true if the correct string is provided and if the flag was
// set to true by calling fail on the dependency beforehand. After simulating a
// crash the flag will be set to false and fail has to be called again for
// another disruption.
func (d *DependencyInterruptOnceOnKeyword) Disrupt(s string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.f && s == d.str {
		d.f = false
		return true
	}
	return false
}

// Disrupt returns true if the correct string is provided, if the flag was set
// to true by calling fail on the dependency and if Disrupt has been called n
// times since fail was called.
func (d *DependencyInterruptAfterNCalls) Disrupt(s string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.f && s == d.str && d.cntr == d.n {
		d.f = false
		d.cntr = 0
		return true
	} else if d.f && s == d.str && d.cntr < d.n {
		d.cntr++
	}
	return false
}

// Fail causes the next call to Disrupt to return true if the correct string is
// provided.
func (d *DependencyInterruptOnceOnKeyword) Fail() {
	d.mu.Lock()
	d.f = true
	d.mu.Unlock()
}

// Disable sets the flag to false to make sure that the dependency won't fail.
func (d *DependencyInterruptOnceOnKeyword) Disable() {
	d.mu.Lock()
	d.f = false
	d.mu.Unlock()
}

// Disrupt will block the scan progress of the hostdb. The scan can be started
// by calling Scan on the dependency.
func (d *DependencyBlockScan) Disrupt(s string) bool {
	if d.c == nil {
		d.c = make(chan struct{})
	}
	if s == "BlockScan" {
		<-d.c
	}
	return false
}

// Disrupt prevents contracts from being recovered in
// threadedContractMaintenance.
func (d *DependencyDisableContractRecovery) Disrupt(s string) bool {
	return s == "DisableContractRecovery"
}

// Disrupt prevents SiafileEntries in the upload code from being closed.
func (d *DependencyDisableCloseUploadEntry) Disrupt(s string) bool {
	return s == "disableCloseUploadEntry"
}

// Disrupt will prevent the fields scanInProgress and atomicRecoveryScanHeight
// from being reset after the scan is done and also prevent automatic contract
// recovery scans from being triggered.
func (d *DependencyDisableRecoveryStatusReset) Disrupt(s string) bool {
	return s == "disableRecoveryStatusReset" || s == "disableAutomaticContractRecoveryScan"
}

// Disrupt will prevent contracts from being renewed.
func (d *DependencyDisableRenewal) Disrupt(s string) bool {
	return s == "disableRenew"
}

// Disrupt will prevent the repair and health loops from running
func (d *DependencyDisableRepairAndHealthLoops) Disrupt(s string) bool {
	return s == "DisableRepairAndHealthLoops"
}

// Scan resumes the blocked scan.
func (d *DependencyBlockScan) Scan() {
	if d.closed {
		return
	}
	close(d.c)
	d.closed = true
}

type (
	// customResolver is a testing resolver which can be created from any
	// lookupIP method.
	customResolver struct {
		lookupIP func(string) ([]net.IP, error)
	}
	// dependencyCustomResolver is a dependency which overrides the Resolver
	// method to return a custom resolver with a specific lookupIP method.
	dependencyCustomResolver struct {
		modules.ProductionDependencies
		lookupIP func(string) ([]net.IP, error)
	}
)

// LookupIP implements the modules.Resolver interface.
func (cr customResolver) LookupIP(host string) ([]net.IP, error) {
	return cr.lookupIP(host)
}

// Disrupt makes sure that hosts which resolve to addresses we can't connect to
// due to the customResolver will be online in the hostdb.
func (d *dependencyCustomResolver) Disrupt(s string) bool {
	return s == "customResolver"
}

// Resolver creates a new custom resolver.
func (d *dependencyCustomResolver) Resolver() modules.Resolver {
	return customResolver{d.lookupIP}
}
