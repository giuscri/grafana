package kindsys

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
	"github.com/grafana/grafana"
	"github.com/grafana/grafana/pkg/cuectx"
	"github.com/grafana/thema"
	tload "github.com/grafana/thema/load"
)

var defaultFramework cue.Value

func init() {
	var err error
	defaultFramework, err = doLoadFrameworkCUE(cuectx.GrafanaCUEContext())
	if err != nil {
		panic(err)
	}
}

var prefix = filepath.Join("/pkg", "kindsys")

func doLoadFrameworkCUE(ctx *cue.Context) (cue.Value, error) {
	var v cue.Value
	var err error

	absolutePath := prefix
	if !filepath.IsAbs(absolutePath) {
		absolutePath, err = filepath.Abs(absolutePath)
		if err != nil {
			return v, err
		}
	}

	bi, err := tload.InstancesWithThema(grafana.CueSchemaFS, absolutePath)
	if err != nil {
		return v, err
	}
	v = ctx.BuildInstance(bi)

	if err = v.Validate(cue.Concrete(false), cue.All()); err != nil {
		return cue.Value{}, fmt.Errorf("coremodel framework loaded cue.Value has err: %w", err)
	}

	return v, nil
}

// CUEFramework returns a cue.Value representing all the kind framework
// raw CUE files.
//
// For low-level use in constructing other types and APIs, while still letting
// us declare all the frameworky CUE bits in a single package. Other Go types
// make the constructs in this value easy to use.
//
// All calling code within grafana/grafana is expected to use Grafana's
// singleton [cue.Context], returned from [cuectx.GrafanaCUEContext]. If nil
// is passed, the singleton will be used.
func CUEFramework(ctx *cue.Context) cue.Value {
	if ctx == nil || ctx == cuectx.GrafanaCUEContext() {
		return defaultFramework
	}
	// Error guaranteed to be nil here because erroring would have caused init() to panic
	v, _ := doLoadFrameworkCUE(ctx) // nolint:errcheck
	return v
}

// ToSomeKindMeta takes a cue.Value expected to represent any one of the kind
// categories, and attempts to extract its metadata into the relevant typed
// struct.
func ToSomeKindMeta(v cue.Value) (SomeKindMeta, error) {
	if !v.Exists() {
		return nil, ErrValueNotExist
	}

	if meta, err := ToKindMeta[RawMeta](v); err == nil {
		return meta, nil
	}
	if meta, err := ToKindMeta[CoreStructuredMeta](v); err == nil {
		return meta, nil
	}
	if meta, err := ToKindMeta[CustomStructuredMeta](v); err == nil {
		return meta, nil
	}
	if meta, err := ToKindMeta[SlotImplMeta](v); err == nil {
		return meta, nil
	}
	return nil, ErrValueNotAKind
}

// ToKindMeta takes a cue.Value expected to represent a kind of the category
// specified by the type parameter and populates the Go type from the cue.Value.
func ToKindMeta[T KindMetas](v cue.Value) (T, error) {
	meta := new(T)
	if !v.Exists() {
		return *meta, ErrValueNotExist
	}

	fw := CUEFramework(v.Context())
	var kdef cue.Value

	anymeta := any(*meta).(SomeKindMeta)
	switch anymeta.(type) {
	case RawMeta:
		kdef = fw.LookupPath(cue.MakePath(cue.Def("Raw")))
	case CoreStructuredMeta:
		kdef = fw.LookupPath(cue.MakePath(cue.Def("CoreStructured")))
	case CustomStructuredMeta:
		kdef = fw.LookupPath(cue.MakePath(cue.Def("CustomStructured")))
	case SlotImplMeta:
		kdef = fw.LookupPath(cue.MakePath(cue.Def("Slot")))
	default:
		// unreachable so long as all the possibilities in KindMetas have switch branches
		panic("unreachable")
	}

	item := v.Unify(kdef)
	if err := item.Validate(cue.Concrete(false), cue.All()); err != nil {
		return *meta, ewrap(item.Err(), ErrValueNotAKind)
	}
	if err := item.Decode(meta); err != nil {
		// Should only be reachable if CUE and Go framework types have diverged
		panic(errors.Details(err, nil))
	}

	return *meta, nil
}

// SomeDecl represents a single kind declaration, having been loaded
// and validated by a func such as [LoadCoreKind].
//
// The underlying type of the Meta field indicates the category of
// kind.
type SomeDecl struct {
	// V is the cue.Value containing the entire Kind declaration.
	V cue.Value
	// Meta contains the kind's metadata settings.
	Meta SomeKindMeta
}

// BindKindLineage binds the lineage for the kind declaration. nil, nil is returned
// for raw kinds.
//
// For kinds with a corresponding Go type, it is left to the caller to associate
// that Go type with the lineage returned from this function by a call to [thema.BindType].
func (decl *SomeDecl) BindKindLineage(rt *thema.Runtime, opts ...thema.BindOption) (thema.Lineage, error) {
	if rt == nil {
		rt = cuectx.GrafanaThemaRuntime()
	}
	switch decl.Meta.(type) {
	case RawMeta:
		return nil, nil
	case CoreStructuredMeta, CustomStructuredMeta, SlotImplMeta:
		return thema.BindLineage(decl.V.LookupPath(cue.MakePath(cue.Str("lineage"))), rt, opts...)
	default:
		panic("unreachable")
	}
}

// IsRaw indicates whether the represented kind is a raw kind.
func (decl *SomeDecl) IsRaw() bool {
	_, is := decl.Meta.(RawMeta)
	return is
}

// IsCoreStructured indicates whether the represented kind is a core structured kind.
func (decl *SomeDecl) IsCoreStructured() bool {
	_, is := decl.Meta.(CoreStructuredMeta)
	return is
}

// IsCustomStructured indicates whether the represented kind is a custom structured kind.
func (decl *SomeDecl) IsCustomStructured() bool {
	_, is := decl.Meta.(CustomStructuredMeta)
	return is
}

// IsSlotImpl indicates whether the represented kind is a slot implementation kind.
func (decl *SomeDecl) IsSlotImpl() bool {
	_, is := decl.Meta.(SlotImplMeta)
	return is
}

// Decl represents a single kind declaration, having been loaded
// and validated by a func such as [LoadCoreKind].
//
// Its type parameter indicates the category of kind.
type Decl[T KindMetas] struct {
	// V is the cue.Value containing the entire Kind declaration.
	V cue.Value
	// Meta contains the kind's metadata settings.
	Meta T
}

// LoadAnyKindFS takes an fs.FS and validates that it contains a valid kind
// definition from any of the kind categories. On success, it returns a
// representation of the entire kind definition contained in the provided kfs.
func LoadAnyKindFS(kfs fs.FS, path string, ctx *cue.Context) (*SomeDecl, error) {
	// TODO use a more genericized loader
	inst, err := tload.InstancesWithThema(kfs, path)
	if err != nil {
		return nil, err
	}

	vk := ctx.BuildInstance(inst)
	if err = vk.Validate(cue.Concrete(false), cue.All()); err != nil {
		return nil, err
	}
	pkd := &SomeDecl{
		V: vk,
	}
	pkd.Meta, err = ToSomeKindMeta(vk)
	if err != nil {
		return nil, err
	}
	return pkd, nil
}

// Some converts the typed Decl to the equivalent typeless SomeDecl.
func (decl *Decl[T]) Some() *SomeDecl {
	return &SomeDecl{
		V:    decl.V,
		Meta: any(decl.Meta).(SomeKindMeta),
	}
}

// LoadCoreKind loads and validates a core kind declaration of the kind category
// indicated by the type parameter. On success, it returns a [Decl] which
// contains the entire contents of the kind declaration.
//
// declpath is the path to the directory containing the core kind declaration,
// relative to the grafana/grafana root. For example, dashboards are in
// "kinds/structured/dashboard".
//
// The bytes containing the .cue file declarations will be retrieved from the
// central embedded FS, [grafana.CueSchemaFS]. If desired (e.g. for testing), an
// optional fs.FS may be provided via the overlay parameter, which will be
// merged over [grafana.CueSchemaFS]. But in all typical circumstances, overlay
// can and should be nil.
//
// This is a low-level function, primarily intended for use in code generation.
// For representations of core kinds that are useful in Go programs at runtime,
// see ["github.com/grafana/grafana/pkg/registry/corekind"].
func LoadCoreKind[T RawMeta | CoreStructuredMeta](declpath string, ctx *cue.Context, overlay fs.FS) (*Decl[T], error) {
	vk, err := cuectx.BuildGrafanaInstance(declpath, "kind", ctx, overlay)
	if err != nil {
		return nil, err
	}
	decl := &Decl[T]{
		V: vk,
	}
	decl.Meta, err = ToKindMeta[T](vk)
	if err != nil {
		return nil, err
	}
	return decl, nil
}
