## Basics

Original Rupack's paper about using independency between inputs to reduce the number of concolic executions: https://dl.acm.org/doi/10.1007/978-3-642-02658-4_41

Another Rupack's paper that is required to better understand the theorem proof in the first one + allows to get a better understanding of flow semantic: https://dl.acm.org/doi/epdf/10.1145/1065010.1065016

There is a useful readme to understand how types are represented in the ssa: https://github.com/golang/example/tree/master/gotypes

The overview of this package is useful to better understand infrastructure around analysis passes: https://pkg.go.dev/golang.org/x/tools/go/analysis 

This package is used to build immediate post-dominators to monitor control flow dependencies: https://github.com/gonum/gonum/tree/master/graph/flow

## Pointers

There are multiple pointers related operations in go ssa:
1. alloc Type instruction - allocates a pointer to a specified type either on heap or in the stack frame. 
2. store instruction (*t0 = t1) 
3. Load instruction (t0 = *t1)

Even though it sounds obvious, it's nice to understand that the pointer itself couldn't be changed to point to an another address. 

Though it doesn't mean that in all cases you could simply figure out to which variable the pointer is gonna point to, because the pointer address could be created by phi instruction. 

So for full correctness we need to maintain an over approximation of the real points-to relation. 
On *t0 = t1 we must update flow for all variables to which t0 can point with flow(t1).
On t0 = *t1 we must update the flow of t0 with flow of all possible values to which t1 may point.

In practice in most of the cases, the variable to which the pointer points could be easily resolved. So now, instead of maintaining a point-to relation approximation we just use a function buildCanonicalName to precisely find the variable to which the pointer points

TODO: does it all works for **int?

## Structs 

In go ssa there is a special type struct (types.Struct) to refer to a field of a struct there is a special instruction FieldAddr (t0 = &t1.field) which return a pointer to the field with a provided name of a provided struct.

The thing above means that structs are treated as single values in general in the ssa. This approach isn't good for our goal of analyzing kubernetes operators, as all their inputs between which we want to look for dependencies are represented as structs.

So instead in the flow analyzer we represent structs as unions of variables, each one corresponding to a struct field leaf. Each such field is represented by structFieldCell. For instance, if you have the following structs:
```
type MyAwesomeStructInner struct {
    a int
    b int
}

type MyAwesomeStructOuter struct {
    inner MyAwesomeStructInner
    c int
}
```

and have a variable t0 of type MyAwesomeStructOuter, instead of having one t0 variable in the flow you are gonna have 3 variables:

```
t0.inner.a, t0.inner.b, t0.c
```

That allows tracking dependencies between struct fields, but requires a lot of precision, as now you almost each instruction you have special cases for structs. For instance, when you perform a load of a struct - you must create new flow variables for each field in the struct and propagate the values to them from corresponding values of the struct being loaded.

## Functions

Functions are not supported yet, but the idea is that points-to and flow are interprocedure, so it's enough to parametrize the function analyzer with starting flow and points-to relations

## Cycles

Cycles are badly supported now. Control flow dependencies may be cyclic, for instance if you have a dependency a -> b -> c, and c -> a. That means that during the analysis if we find a backward edge we must update the dependencies and check all the blocks that depends on the updated one. Now it isn't done, it should be done like some kind of fixed-point computation probably, need to think a bit more about this

## Correctness

The original paper (https://dl.acm.org/doi/10.1007/978-3-642-02658-4_41) proposes a very clear idea and a correctness criteria for the partition being correct. It also suggests a way how to calculate this partition based on the "flow" calculated for all variables.

The problem is that the paper doesn't provide clear flow(x) semantics, which makes it harder to:
1. Understand why the proposed in the paper way of flow calculation is correct (spoiler, there is a typo in the program pseudocode)
2. Understand how to extend the flow calculation for pointers and other instructions (such as phi nodes and etc) to stay inside the main theorem assumption.

One may say that to understand it it's enough to understand the proof of the main theorem, but it's not truth in that case.

The correctness theorem statement is the following: if a location x is reachable with an execution with all variables being symbolic, than it's also reachable with the partition calculated by the FlowTest algorithm (the partition calculation algorithm from the paper).

The proof itself is a contradiction proof, that essentially says the following, let's take a path to location x from the execution in which all variables are concolic, now let's drop all operations along this path that doesn't affect the reachability of location x. This is called path slice (and described in the second paper, see Basics section above). And now you must show that all variables that are present in any operations in the path slice are gonna be in the same block of the partition.

The last part is where the definition of flow becomes important, as the block merge is based on the calculated flow and this part is skipped in the original proof. 

Though, the second paper that explains how to build such a path slice significantly helps, because if you understand how the path slices are build you understand what variables you are gonna see in them.

So essentially the path slice building algorithm just takes a path to the location with all the operations, and then drop the ones that are unrelated to reachability of the last location. So you essentially drop ifs that you post dominate and that doesn't affect values on which ifs that you don't postdominate depend. 

It's also interesting that both papers have no idea whether the proposed partition is minimal, and whether the proposed path slice is minimal. Though, it's obvious that you could really find optimal one with static analysis (you couldn't calculate the real points-to statically)
