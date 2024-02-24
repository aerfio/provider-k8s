//go:build generate
// +build generate

/*
copied from provider-helm to fix angryjet (installed into ./bin dir) error:

bin/angryjet-v0.0.0-20230925130601-628280f8bf79 generate-methodsets --header-file=./hack/boilerplate.go.txt ./...
panic: runtime error: invalid memory address or nil pointer dereference [recovered]
        panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0xbebe05e]

goroutine 168 [running]:
go/types.(*Checker).handleBailout(0xc000aca000, 0xc000949c60)
        /usr/local/opt/go/libexec/src/go/types/check.go:367 +0x88
panic({0xc0ffd80?, 0xc3920b0?})
        /usr/local/opt/go/libexec/src/runtime/panic.go:770 +0x132
go/types.(*StdSizes).Sizeof(0x0, {0xc15db00, 0xc3962a0})
        /usr/local/opt/go/libexec/src/go/types/sizes.go:228 +0x31e
go/types.(*Config).sizeof(...)
        /usr/local/opt/go/libexec/src/go/types/sizes.go:333
go/types.representableConst.func1({0xc15db00?, 0xc3962a0?})
        /usr/local/opt/go/libexec/src/go/types/const.go:76 +0x9e
go/types.representableConst({0xc15f838, 0xc389700}, 0xc000aca000, 0xc3962a0, 0xc000948878)
        /usr/local/opt/go/libexec/src/go/types/const.go:92 +0x192
go/types.(*Checker).representation(0xc000aca000, 0xc00080cb40, 0xc3962a0)
        /usr/local/opt/go/libexec/src/go/types/const.go:256 +0x65
go/types.(*Checker).implicitTypeAndValue(0xc000aca000, 0xc00080cb40, {0xc15db00, 0xc3962a0})
        /usr/local/opt/go/libexec/src/go/types/expr.go:375 +0x30d
go/types.(*Checker).assignment(0xc000aca000, 0xc00080cb40, {0xc15db00, 0xc3962a0}, {0xc037418, 0x10})
        /usr/local/opt/go/libexec/src/go/types/assignments.go:52 +0x2e5
go/types.(*Checker).initVar(0xc000aca000, 0xc000a97b00, 0xc00080cb40, {0xc037418, 0x10})
        /usr/local/opt/go/libexec/src/go/types/assignments.go:163 +0x41e
go/types.(*Checker).initVars(0xc000aca000, {0xc000aa2198, 0x1, 0xc000100a18?}, {0xc00038a200, 0xc000ab0de8?, 0x8ff1a46b8ae4e0cd?}, {0xc15ebb8, 0xc000574c80})
        /usr/local/opt/go/libexec/src/go/types/assignments.go:382 +0x638
go/types.(*Checker).stmt(0xc000aca000, 0x0, {0xc15ebb8, 0xc000574c80})
        /usr/local/opt/go/libexec/src/go/types/stmt.go:524 +0x1fc5
go/types.(*Checker).stmtList(0xc000aca000, 0x0, {0xc00038a210?, 0x0?, 0x0?})
        /usr/local/opt/go/libexec/src/go/types/stmt.go:121 +0x85
go/types.(*Checker).funcBody(0xc000aca000, 0xc15db00?, {0xc000a881f4?, 0xc3962a0?}, 0xc00080cac0, 0xc000a9c6c0, {0x0?, 0x0?})
        /usr/local/opt/go/libexec/src/go/types/stmt.go:41 +0x331
go/types.(*Checker).funcDecl.func1()
        /usr/local/opt/go/libexec/src/go/types/decl.go:852 +0x3a
go/types.(*Checker).processDelayed(0xc000aca000, 0x0)
        /usr/local/opt/go/libexec/src/go/types/check.go:467 +0x162
go/types.(*Checker).checkFiles(0xc000aca000, {0xc00038a030, 0x2, 0x2})
        /usr/local/opt/go/libexec/src/go/types/check.go:411 +0x1cc
go/types.(*Checker).Files(...)
        /usr/local/opt/go/libexec/src/go/types/check.go:372
golang.org/x/tools/go/packages.(*loader).loadPackage(0xc00019a000, 0xc00078d420)
        /Users/aerfio/go/pkg/mod/golang.org/x/tools@v0.1.12/go/packages/packages.go:1001 +0x76f
golang.org/x/tools/go/packages.(*loader).loadRecursive.func1()
        /Users/aerfio/go/pkg/mod/golang.org/x/tools@v0.1.12/go/packages/packages.go:838 +0x1a9
sync.(*Once).doSlow(0x0?, 0x0?)
        /usr/local/opt/go/libexec/src/sync/once.go:74 +0xc2
sync.(*Once).Do(...)
        /usr/local/opt/go/libexec/src/sync/once.go:65
golang.org/x/tools/go/packages.(*loader).loadRecursive(0x0?, 0x0?)
        /Users/aerfio/go/pkg/mod/golang.org/x/tools@v0.1.12/go/packages/packages.go:826 +0x4a
golang.org/x/tools/go/packages.(*loader).loadRecursive.func1.1(0x0?)
        /Users/aerfio/go/pkg/mod/golang.org/x/tools@v0.1.12/go/packages/packages.go:833 +0x26
created by golang.org/x/tools/go/packages.(*loader).loadRecursive.func1 in goroutine 161
        /Users/aerfio/go/pkg/mod/golang.org/x/tools@v0.1.12/go/packages/packages.go:832 +0x94
make: *** [generate] Error 2

*/

/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// NOTE: See the below link for details on what is happening here.
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

// Remove existing CRDs
//go:generate rm -rf ../package/crds

// Generate deepcopy methodsets and CRD manifests
//go:generate go run -tags generate sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../hack/boilerplate.go.txt paths=./... crd:crdVersions=v1 output:artifacts:config=../package/crds

// Generate crossplane-runtime methodsets (resource.Claim, etc)
//go:generate go run -tags generate github.com/crossplane/crossplane-tools/cmd/angryjet generate-methodsets --header-file=../hack/boilerplate.go.txt ./...

package apis

import (
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen" //nolint:typecheck

	_ "github.com/crossplane/crossplane-tools/cmd/angryjet" //nolint:typecheck
)
