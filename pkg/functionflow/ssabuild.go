package functionflow

import (
	"errors"
	"fmt"
	"go/types"
	"os"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func buildPackage(pathToPkg string) (*packages.Package, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.LoadAllSyntax,
	}, pathToPkg)
	if err != nil {
		return nil, fmt.Errorf("can't load package for analysis: %w", err)
	}
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("package load returned more than one pkg: %d", len(pkgs))
	}

	return pkgs[0], nil
}

// pkgErrors collects the load and type errors of root and of all its dependencies.
// ssautil.AllPackages returns a nil package when its source package is ill-typed,
// and the reason is only available in packages.Package.Errors.
func pkgErrors(originalPkg *packages.Package) error {
	var errs []error
	packages.Visit([]*packages.Package{originalPkg}, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			errs = append(errs, e)
		}
	})

	return errors.Join(errs...)
}

func buildSSA(pkg *packages.Package, funcName string) (*ssa.Function, error) {
	prog, ssaPkgs := ssautil.AllPackages([]*packages.Package{pkg}, 0)
	prog.Build()
	var ssaFunc *ssa.Function

	for _, ssaPkg := range ssaPkgs {
		if ssaPkg == nil {
			return nil, fmt.Errorf("can't compile package %s: %w", pkg.PkgPath, pkgErrors(pkg))
		}

		if res := findFunc(prog, ssaPkg, funcName); res != nil {
			ssaFunc = res
		}
	}

	if ssaFunc == nil {
		return nil, fmt.Errorf("can't find ssa representation of the given func: %s", funcName)
	}

	return ssaFunc, nil
}

// findFunc looks for a package-level function with the given name and, if there is none,
// for a method with that name on one of the package's types. The name is assumed to be
// exported and unique in the package.
func findFunc(prog *ssa.Program, ssaPkg *ssa.Package, name string) *ssa.Function {
	if fn := ssaPkg.Func(name); fn != nil {
		return fn
	}

	for _, member := range ssaPkg.Members {
		typ, ok := member.(*ssa.Type)
		if !ok {
			continue
		}
		// Generic types have no method set until they are instantiated, so skip generic types
		if named, ok := typ.Type().(*types.Named); ok && named.TypeParams().Len() > 0 {
			continue
		}

		// The method set of *T contains the methods with both value and pointer receivers
		methodSet := prog.MethodSets.MethodSet(types.NewPointer(typ.Type()))
		selection := methodSet.Lookup(ssaPkg.Pkg, name)
		if selection == nil {
			continue
		}
		if fn := prog.MethodValue(selection); fn != nil {
			return fn
		}
	}

	return nil
}

// dumpSSA writes the SSA form of fn to the file at filePath, truncating it if it exists.
func dumpSSA(fn *ssa.Function, filePath string) (err error) {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("can't create ssa dump file %s: %w", filePath, err)
	}
	defer func() {
		// A failed Close can mean the buffered data was not written
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("can't close ssa dump file %s: %w", filePath, closeErr)
		}
	}()

	if _, err := fn.WriteTo(file); err != nil {
		return fmt.Errorf("can't write ssa to %s: %w", filePath, err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("can't sync ssa dump file %s: %w", filePath, err)
	}

	return nil
}
