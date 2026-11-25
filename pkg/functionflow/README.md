## Basics

Rupak Majumdar's original paper on using independence between inputs to reduce the number of concolic executions: https://dl.acm.org/doi/10.1007/978-3-642-02658-4_41

Another paper by Rupak Majumdar, needed to better understand the proof of the theorem in the first one; it also gives a better understanding of flow semantics: https://dl.acm.org/doi/epdf/10.1145/1065010.1065016

There is a useful README for understanding how types are represented in SSA: https://github.com/golang/example/tree/master/gotypes

The overview of this package is useful for understanding the infrastructure around analysis passes: https://pkg.go.dev/golang.org/x/tools/go/analysis 

This package is used to build immediate post-dominators to monitor control flow dependencies: https://github.com/gonum/gonum/tree/master/graph/flow

## Pointers

There are several pointer-related operations in Go SSA:
1. Alloc instruction (alloc Type) - allocates a value of the specified type, either on the heap or in the stack frame, and returns a pointer to it. 
2. Store instruction (*t0 = t1) 
3. Load instruction (t0 = *t1)

Even though it sounds obvious, it's worth noting that a pointer itself can't be changed to point to another address. 

However, this doesn't mean you can always easily figure out which variable a pointer points to, because the pointer may be produced by a phi instruction. 

So for full correctness, we need to maintain an over-approximation of the real points-to relation. 
On *t0 = t1, we must update the flow of every variable that t0 may point to with flow(t1).
On t0 = *t1, we must update the flow of t0 with the flow of every value that t1 may point to.

In practice, in most cases the variable a pointer points to can be easily resolved, though this isn't true in general. So for now, instead of maintaining an approximation of the points-to relation, we just use the buildCanonicalName function to find exactly which variable the pointer points to.

TODO: does all of this work for **int?

## Structs 

Go SSA has a dedicated struct type (types.Struct). To refer to a field of a struct, there is a special instruction, FieldAddr (t0 = &t1.field), which returns a pointer to the named field of the given struct.

This means that, in general, SSA treats structs as single values. That doesn't suit our goal of analyzing Kubernetes operators, because all the inputs whose dependencies we want to find are represented as structs.

So instead, the flow analyzer represents a struct as a union of variables, each corresponding to a leaf field of the struct. Each such field is represented by a structFieldCell. For instance, given the following structs:
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

and a variable t0 of type MyAwesomeStructOuter, the flow will contain 3 variables instead of a single variable t0:

```
t0.inner.a, t0.inner.b, t0.c
```

This allows tracking dependencies between struct fields, but requires a lot of precision, since almost every instruction now has special cases for structs. For instance, when you load a struct, you must create new flow variables for each of its fields and propagate values to them from the corresponding fields of the struct being loaded.

## Functions

Functions aren't supported yet, but the idea is that points-to and flow are interprocedural, so it's enough to parameterize the function analyzer with the initial flow and points-to relations.

## Cycles

Cycles are poorly supported right now. Control-flow dependencies may be cyclic, for instance if you have dependencies a -> b -> c and c -> a. This means that if the analysis finds a back edge, we must update the dependencies and recheck all blocks that depend on the updated one. This isn't done yet; it should probably be implemented as some kind of fixed-point computation. This needs more thought.

## Correctness

The original paper (https://dl.acm.org/doi/10.1007/978-3-642-02658-4_41) proposes a very clear idea and a correctness criterion for the partition. It also suggests a way to calculate this partition based on the "flow" calculated for all variables.

The problem is that the paper doesn't provide clear flow(x) semantics, which makes it harder to:
1. Understand why the flow calculation proposed in the paper is correct (spoiler: there is a typo in the pseudocode)
2. Understand how to extend the flow calculation to pointers and other instructions (such as phi nodes) while staying within the assumptions of the main theorem.

One might say that understanding the proof of the main theorem is enough, but that isn't true in this case.

The correctness theorem states the following: if a location x is reachable by an execution in which all variables are symbolic, then it is also reachable with the partition calculated by the FlowTest algorithm (the partition calculation algorithm from the paper).

The proof is by contradiction and essentially goes as follows: take a path to location x from the execution in which all variables are concolic, then drop all operations along this path that don't affect the reachability of x. The result is called a path slice (described in the second paper; see the Basics section above). Now you must show that all variables that appear in any operation of the path slice end up in the same block of the partition.

The last part is where the definition of flow becomes important, as the block merge is based on the calculated flow and this part is skipped in the original proof. 

However, the second paper, which explains how to build such a path slice, helps significantly: once you understand how path slices are built, you understand which variables will appear in them.

Essentially, the path slicing algorithm takes a path to the location with all its operations and drops the ones that are unrelated to the reachability of the final location. In other words, you drop the ifs that the location post-dominates and that don't affect the values that the non-post-dominated ifs depend on. 

It's also interesting that neither paper says whether the proposed partition or the proposed path slice is minimal. However, it's clear that you can't find the optimal one with static analysis in general (you can't compute the real points-to relation statically).
