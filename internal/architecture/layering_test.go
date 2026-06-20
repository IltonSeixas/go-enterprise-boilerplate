// Package architecture enforces the Clean Architecture dependency rule from
// ADR-0001 by walking the real import graph. See ADR-0006.
package architecture

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const modulePath = "github.com/IltonSeixas/go-enterprise-boilerplate"

var infrastructurePackages = []string{
	"github.com/gin-gonic/gin",
	"github.com/jackc/pgx",
	"github.com/redis/go-redis",
	"github.com/golang-jwt/jwt",
	"github.com/alicebob/miniredis",
	"go.opentelemetry.io",
	"github.com/prometheus/client_golang",
	"go.uber.org/zap",
	"github.com/spf13/viper",
	"golang.org/x/crypto",
	"google.golang.org/grpc",
	"google.golang.org/protobuf",
	modulePath + "/internal/infrastructure",
	modulePath + "/internal/interface",
}

func loadModulePackages(t *testing.T, pattern string) []*packages.Package {
	t.Helper()
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		t.Fatalf("failed to load packages %s: %v", pattern, err)
	}
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			t.Fatalf("package load error in %s: %v", pkg.PkgPath, e)
		}
	}
	return pkgs
}

func violations(pkgs []*packages.Package, forbidden []string) []string {
	var found []string
	for _, pkg := range pkgs {
		for imp := range pkg.Imports {
			for _, f := range forbidden {
				if imp == f || strings.HasPrefix(imp, f+"/") {
					found = append(found, fmt.Sprintf("%s imports %s", pkg.PkgPath, imp))
				}
			}
		}
	}
	return found
}

func TestDomainMustNotDependOnInfrastructureOrApplication(t *testing.T) {
	pkgs := loadModulePackages(t, modulePath+"/internal/domain/...")
	forbidden := append([]string{modulePath + "/internal/application"}, infrastructurePackages...)
	if v := violations(pkgs, forbidden); len(v) > 0 {
		t.Fatalf("domain/ must not depend on application/ or infrastructure crates — found:\n%s", strings.Join(v, "\n"))
	}
}

func TestApplicationMustNotDependOnInfrastructure(t *testing.T) {
	pkgs := loadModulePackages(t, modulePath+"/internal/application/...")
	if v := violations(pkgs, infrastructurePackages); len(v) > 0 {
		t.Fatalf("application/ must stay portable across infrastructure adapters — found:\n%s", strings.Join(v, "\n"))
	}
}
