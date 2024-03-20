// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package extensions

import (
	"sync"
)

// Ensure, that ExtensionMock does implement Extension.
// If this is not the case, regenerate this file with moq.
var _ Extension = &ExtensionMock{}

// ExtensionMock is a mock implementation of Extension.
//
//	func TestSomethingThatUsesExtension(t *testing.T) {
//
//		// make and configure a mocked Extension
//		mockedExtension := &ExtensionMock{
//			CurrentVersionFunc: func() string {
//				panic("mock out the CurrentVersion method")
//			},
//			IsBinaryFunc: func() bool {
//				panic("mock out the IsBinary method")
//			},
//			IsLocalFunc: func() bool {
//				panic("mock out the IsLocal method")
//			},
//			IsPinnedFunc: func() bool {
//				panic("mock out the IsPinned method")
//			},
//			LatestVersionFunc: func() string {
//				panic("mock out the LatestVersion method")
//			},
//			NameFunc: func() string {
//				panic("mock out the Name method")
//			},
//			OwnerFunc: func() string {
//				panic("mock out the Owner method")
//			},
//			PathFunc: func() string {
//				panic("mock out the Path method")
//			},
//			URLFunc: func() string {
//				panic("mock out the URL method")
//			},
//			UpdateAvailableFunc: func() bool {
//				panic("mock out the UpdateAvailable method")
//			},
//		}
//
//		// use mockedExtension in code that requires Extension
//		// and then make assertions.
//
//	}
type ExtensionMock struct {


	// IsBinaryFunc mocks the IsBinary method.
	IsBinaryFunc func() bool

	// IsLocalFunc mocks the IsLocal method.
	IsLocalFunc func() bool

	// IsPinnedFunc mocks the IsPinned method.
	IsPinnedFunc func() bool

	// LatestVersionFunc mocks the LatestVersion method.
	LatestVersionFunc func() string

	// NameFunc mocks the Name method.
	NameFunc func() string

	// OwnerFunc mocks the Owner method.
	OwnerFunc func() string

	// PathFunc mocks the Path method.
	PathFunc func() string

	// URLFunc mocks the URL method.
	URLFunc func() string

	// UpdateAvailableFunc mocks the UpdateAvailable method.
	UpdateAvailableFunc func() bool

	// calls tracks calls to the methods.
	calls struct {
		// CurrentVersion holds details about calls to the CurrentVersion method.
		CurrentVersion []struct {
		}
		// IsBinary holds details about calls to the IsBinary method.
		IsBinary []struct {
		}
		// IsLocal holds details about calls to the IsLocal method.
		IsLocal []struct {
		}
		// IsPinned holds details about calls to the IsPinned method.
		IsPinned []struct {
		}
		// LatestVersion holds details about calls to the LatestVersion method.
		LatestVersion []struct {
		}
		// Name holds details about calls to the Name method.
		Name []struct {
		}
		// Owner holds details about calls to the Owner method.
		Owner []struct {
		}
		// Path holds details about calls to the Path method.
		Path []struct {
		}
		// URL holds details about calls to the URL method.
		URL []struct {
		}
		// UpdateAvailable holds details about calls to the UpdateAvailable method.
		UpdateAvailable []struct {
		}
	}
	lockCurrentVersion  sync.RWMutex
	lockIsBinary        sync.RWMutex
	lockIsLocal         sync.RWMutex
	lockIsPinned        sync.RWMutex
	lockLatestVersion   sync.RWMutex
	lockName            sync.RWMutex
	lockOwner           sync.RWMutex
	lockPath            sync.RWMutex
	lockURL             sync.RWMutex
	lockUpdateAvailable sync.RWMutex
}

// CurrentVersion calls CurrentVersionFunc.
func (mock *ExtensionMock) CurrentVersion() string {
	if mock.CurrentVersionFunc == nil {
		panic("ExtensionMock.CurrentVersionFunc: method is nil but Extension.CurrentVersion was just called")
	}
	callInfo := struct {
	}{}
	mock.lockCurrentVersion.Lock()
	mock.calls.CurrentVersion = append(mock.calls.CurrentVersion, callInfo)
	mock.lockCurrentVersion.Unlock()
	return mock.CurrentVersionFunc()
}

// CurrentVersionCalls gets all the calls that were made to CurrentVersion.
// Check the length with:
//
//	len(mockedExtension.CurrentVersionCalls())
func (mock *ExtensionMock) CurrentVersionCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockCurrentVersion.RLock()
	calls = mock.calls.CurrentVersion
	mock.lockCurrentVersion.RUnlock()
	return calls
}

// IsBinary calls IsBinaryFunc.
func (mock *ExtensionMock) IsBinary() bool {
	if mock.IsBinaryFunc == nil {
		panic("ExtensionMock.IsBinaryFunc: method is nil but Extension.IsBinary was just called")
	}
	callInfo := struct {
	}{}
	mock.lockIsBinary.Lock()
	mock.calls.IsBinary = append(mock.calls.IsBinary, callInfo)
	mock.lockIsBinary.Unlock()
	return mock.IsBinaryFunc()
}

// IsBinaryCalls gets all the calls that were made to IsBinary.
// Check the length with:
//
//	len(mockedExtension.IsBinaryCalls())
func (mock *ExtensionMock) IsBinaryCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockIsBinary.RLock()
	calls = mock.calls.IsBinary
	mock.lockIsBinary.RUnlock()
	return calls
}

// IsLocal calls IsLocalFunc.
func (mock *ExtensionMock) IsLocal() bool {
	if mock.IsLocalFunc == nil {
		panic("ExtensionMock.IsLocalFunc: method is nil but Extension.IsLocal was just called")
	}
	callInfo := struct {
	}{}
	mock.lockIsLocal.Lock()
	mock.calls.IsLocal = append(mock.calls.IsLocal, callInfo)
	mock.lockIsLocal.Unlock()
	return mock.IsLocalFunc()
}

// IsLocalCalls gets all the calls that were made to IsLocal.
// Check the length with:
//
//	len(mockedExtension.IsLocalCalls())
func (mock *ExtensionMock) IsLocalCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockIsLocal.RLock()
	calls = mock.calls.IsLocal
	mock.lockIsLocal.RUnlock()
	return calls
}

// IsPinned calls IsPinnedFunc.
func (mock *ExtensionMock) IsPinned() bool {
	if mock.IsPinnedFunc == nil {
		panic("ExtensionMock.IsPinnedFunc: method is nil but Extension.IsPinned was just called")
	}
	callInfo := struct {
	}{}
	mock.lockIsPinned.Lock()
	mock.calls.IsPinned = append(mock.calls.IsPinned, callInfo)
	mock.lockIsPinned.Unlock()
	return mock.IsPinnedFunc()
}

// IsPinnedCalls gets all the calls that were made to IsPinned.
// Check the length with:
//
//	len(mockedExtension.IsPinnedCalls())
func (mock *ExtensionMock) IsPinnedCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockIsPinned.RLock()
	calls = mock.calls.IsPinned
	mock.lockIsPinned.RUnlock()
	return calls
}

// LatestVersion calls LatestVersionFunc.
func (mock *ExtensionMock) LatestVersion() string {
	if mock.LatestVersionFunc == nil {
		panic("ExtensionMock.LatestVersionFunc: method is nil but Extension.LatestVersion was just called")
	}
	callInfo := struct {
	}{}
	mock.lockLatestVersion.Lock()
	mock.calls.LatestVersion = append(mock.calls.LatestVersion, callInfo)
	mock.lockLatestVersion.Unlock()
	return mock.LatestVersionFunc()
}

// LatestVersionCalls gets all the calls that were made to LatestVersion.
// Check the length with:
//
//	len(mockedExtension.LatestVersionCalls())
func (mock *ExtensionMock) LatestVersionCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockLatestVersion.RLock()
	calls = mock.calls.LatestVersion
	mock.lockLatestVersion.RUnlock()
	return calls
}

// Name calls NameFunc.
func (mock *ExtensionMock) Name() string {
	if mock.NameFunc == nil {
		panic("ExtensionMock.NameFunc: method is nil but Extension.Name was just called")
	}
	callInfo := struct {
	}{}
	mock.lockName.Lock()
	mock.calls.Name = append(mock.calls.Name, callInfo)
	mock.lockName.Unlock()
	return mock.NameFunc()
}

// NameCalls gets all the calls that were made to Name.
// Check the length with:
//
//	len(mockedExtension.NameCalls())
func (mock *ExtensionMock) NameCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockName.RLock()
	calls = mock.calls.Name
	mock.lockName.RUnlock()
	return calls
}

// Owner calls OwnerFunc.
func (mock *ExtensionMock) Owner() string {
	if mock.OwnerFunc == nil {
		panic("ExtensionMock.OwnerFunc: method is nil but Extension.Owner was just called")
	}
	callInfo := struct {
	}{}
	mock.lockOwner.Lock()
	mock.calls.Owner = append(mock.calls.Owner, callInfo)
	mock.lockOwner.Unlock()
	return mock.OwnerFunc()
}

// OwnerCalls gets all the calls that were made to Owner.
// Check the length with:
//
//	len(mockedExtension.OwnerCalls())
func (mock *ExtensionMock) OwnerCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockOwner.RLock()
	calls = mock.calls.Owner
	mock.lockOwner.RUnlock()
	return calls
}

// Path calls PathFunc.
func (mock *ExtensionMock) Path() string {
	if mock.PathFunc == nil {
		panic("ExtensionMock.PathFunc: method is nil but Extension.Path was just called")
	}
	callInfo := struct {
	}{}
	mock.lockPath.Lock()
	mock.calls.Path = append(mock.calls.Path, callInfo)
	mock.lockPath.Unlock()
	return mock.PathFunc()
}

// PathCalls gets all the calls that were made to Path.
// Check the length with:
//
//	len(mockedExtension.PathCalls())
func (mock *ExtensionMock) PathCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockPath.RLock()
	calls = mock.calls.Path
	mock.lockPath.RUnlock()
	return calls
}

// URL calls URLFunc.
func (mock *ExtensionMock) URL() string {
	if mock.URLFunc == nil {
		panic("ExtensionMock.URLFunc: method is nil but Extension.URL was just called")
	}
	callInfo := struct {
	}{}
	mock.lockURL.Lock()
	mock.calls.URL = append(mock.calls.URL, callInfo)
	mock.lockURL.Unlock()
	return mock.URLFunc()
}

// URLCalls gets all the calls that were made to URL.
// Check the length with:
//
//	len(mockedExtension.URLCalls())
func (mock *ExtensionMock) URLCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockURL.RLock()
	calls = mock.calls.URL
	mock.lockURL.RUnlock()
	return calls
}

// UpdateAvailable calls UpdateAvailableFunc.
func (mock *ExtensionMock) UpdateAvailable() bool {
	if mock.UpdateAvailableFunc == nil {
		panic("ExtensionMock.UpdateAvailableFunc: method is nil but Extension.UpdateAvailable was just called")
	}
	callInfo := struct {
	}{}
	mock.lockUpdateAvailable.Lock()
	mock.calls.UpdateAvailable = append(mock.calls.UpdateAvailable, callInfo)
	mock.lockUpdateAvailable.Unlock()
	return mock.UpdateAvailableFunc()
}

// UpdateAvailableCalls gets all the calls that were made to UpdateAvailable.
// Check the length with:
//
//	len(mockedExtension.UpdateAvailableCalls())
func (mock *ExtensionMock) UpdateAvailableCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockUpdateAvailable.RLock()
	calls = mock.calls.UpdateAvailable
	mock.lockUpdateAvailable.RUnlock()
	return calls
}
