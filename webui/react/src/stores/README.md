# State Store


goal: to make it so that any data that is either
- relevant across multiple pages, OR
- of small size and/or slow changing, OR
- persisted from the UI

should:
- be easily accessible with minimal boilerplate
- trigger re-renders when the data changes
- without triggering unnecessary re-renders
- without causing redundant API calls


we would like to have a store to be able to have a single, reliable, developer friendly source of truth for application state.

honing in on state a bit, there are broadly 4 different kinds of state in our webui:
  - "small" server state: ✅ goes in store  
    (basically everything in our current store contexts)  
    - how many agents are connected  
    - how many resource pools are available  
    - active tasks  
    - workspaces  
    - roles  
    - users  
    - tasks  
  - "large" server state: 🚫 doesn't go in store (but maybe could)  
    - all the possible combinations of Record<filters, experiments>  
    - all the metrics for all the experiments  
  - persisted ui state: ✅ goes in store  
    (basically, everything for which we currently useSettings)  
    - the filters/sorting a user has applied for a given project  
    - persisted user inputs (maybe), such as tags and descriptions a user adds to an experiment  
  - ephemeral ui state: 🚫 doesnt go in store  
    - this includes user inputs for a form, etc.  

previously, we had all of the small server state in a single store context. but this caused peformance and ui issues because of the behavior of React context, where any changes to anything in the context would cause unnecessary rerenders in context consumers. we split the different aspects of the store out into separate contexts to avoid this issue. however, the current solution is not very ergonomic and requires some boilerplate to consume. furthermore, we have had lingering issues around application loading state and data inconsistency between pages.

there exist observable libraries that are specifically designed to provide a unified store that is well optimized for avoiding unnecessary re-renders, and simultaneously being very easy to use.

after an initial evaluation, there were two main candidates for the library:
 - mobx
 - micro-observables


## mobx
lets say you just wanted to do a state store for agents and resource pools. in the webui, we have places where we use agents/resource pools directly, and we also have places where we use derived/computed data, such as:
 - `clusterOverview`, which is derived from `agents` and looks something like:
```tsx
{ 
    ResourceType.CPU : {allocation: 1, available: 1, total: 2},
    ResourceType.CUDA : {allocation: 39, available: 1: total: 40}
}
```
 - `clusterStatus`, a percentage string representing the allocation of the cluster, which is dervived from agents, resource pools, AND cluster overview. in mobx, we could implement such a store like this:

```tsx
class StoreService {
  agents: Loadable<Agent[]> = NotLoaded;
  resourcePools: Loadable<ResourcePool[]> = NotLoaded;

  constructor() {
    makeAutoObservable(this)
  }

  get clusterOverview(): Loadable<ClusterOverview> {
    return clusterOverviewDerivation(this.agents)
  }

  get clusterStatus(): string | undefined {
    return clusterStatusDerivation(
      this.clusterOverview,
      this.resourcePools,
      this.agents
    )
  }

  /*  logic to fetch/poll agents and resource pools */
}

export const storeService = new StoreService()
```

at this point `storeService` is our store. you can think of it like a JS object where you can subscribe to changes for the individual keys. what this 'subscribing' looks like can vary, but here is an example for a react component, like  `<Cluster />` 

```tsx
import { storeService } from './store'
import { observer } from 'mobx-react'


export const Cluster = observer(() => {
  const agents = storeService.agents;
  const resourcePools = storeService.resourcePools
  const clusterOverview = storeService.clusterOverview // returns the result of the get fn
  const clusterStatus = storeService.clusterStatus // returns the result of the get fn

  return (
      <div>
        <p>{agents}</p>
        <p>{resourcePools}</p>
        <p>{clusterOverview}</p>
        <p>{clusterStatus}</p>
      </div>
  )
})
```
wrapping a react component in the `observer` functioncauses the component to 'observe' all the observables it accessed, and re-render whenever they change- no props or dependency arrays needed! (at least for this simple case).

the really cool thing though is that, even though `get clusterOverview()` is a function, it is NOT evaluated every time you do `store.clusterOverview`, only ONCE for every time the observables it accesses change. in this way all the data is available just as easily as it would be if you were eagerly computing everything all of the time, while actually having zero redundancy in the computations. likewise if you never `get clusterOverview()`, the function is never called.

the mobx docs describe the strategy like this:
>	Make sure that everything that can be derived from the application state, will be derived. Automatically.

## micro-observables

`micro-observables` accomplishes basically the same thing, although with a slightly different style. one that is a tad more verbose, react-y, and explicit:


```tsx
class StoreService {
  agents: Loadable<Agent[]> = observable(NotLoaded);
  resourcePools: Loadable<ResourcePool[]> = observable(NotLoaded);

  clusterOverview = this.agents.select(clusterOverviewDerivation)

  clusterStatus = Observable.select(
    [this.clusterOverview, this.resourcePools, this.agents],
    clusterStatusDerivation
  )

  /*  logic to fetch/poll agents and resource pools */
}

export const storeService = new StoreService()
```

here we are using `Observable.select(derivationFn)` for our derived data instead of a JS `get`er. It's a bit more clear what is going on since getters aren't magically having their behavior changed.

the way we consume observables in `micro-observables` is also more explicit:

```tsx
import { storeService } from './store'
import { useObservable, Observable } from 'micro-observables'


export const Cluster = observer(() => {
  const agents = useObservable(storeService.agents);
  const resourcePools = useObservable(storeService.resourcePools)
  const clusterOverview = useObservable(storeService.clusterOverview)

  return (
      <div>
        <p>{agents}</p>
        <p>{resourcePools}</p>
        <p>{clusterOverview}</p>
      </div>
  )
})
```

this has the advantage that if you forget to `useObservable`, it breaks in an obvious, type-enforced way: if you say `const agents = storeService.agents`, typescript will tell you "I do not think it means what you think it means," whereas `mobx` will simply not re-render your component if you don't wrap in in `observer` (the `mobx` docs claim that is the number one cause of expected re-renders not happening)

there are more filled out versions of each of these stores here in the repo. the `micro-observable` one is currently active. to switch to the `mobx` store, just do a find and replace for `stores/micro-observables` -> `stores/mobx`. to test things out, you can go to and from the cluster page and resource pool details/job queue


this `StoreService` could grow to encompass *everything* mentioned above for inclusion in the store, along with all the user settings. having 



on balance, my recommendation would be to use `micro-observables` as i believe it is more explicit and less error prone. 

also, even though `mobx` has a large community and significant adoption, a lot of the examples you online use the experimental decorators feature. this is because, as of the previous version of `mobx`, it looked like it was imminent that decorators would be a part of JS. however, the decorators proposal has stagnated and the future of it is unclear, and they are no longer permitted for the newest version of mobx. so this can make for some confusion, since most of the examples online are not valid javascript (and won't be anytime soon).

overall, the
- lack of explicitness
- relative opaqueness/error-proneness compared to `micro-observables`
- non-JSness of online examples

offset the developer-friendliness that would otherwise come from a library with a large community following, hence the recommendation for `micro-observables`.

on top of that `micro-observables` is a super clean, super small codebase with an API you can easily wrap your head around in its entirety.

## alternatives considered

rxjs is another library for observables, and is the most fully featured. but is a lot more opaque and complex, and it does not appear to offer anything that we would need in an initial version of an observable state store. on top of that, members of the team have used it before and found it lacking, so it would not be my recommendation. `micro-observables` replicates the most useful part of the `rxjs` API with a cleaner syntax, so if we started with `micro-observables`, we would be able to easily transition `rxjs` if we ever found its additional features necessary.

recoil: is another library that has been floated as a candidate for our state management library, but it is currently in experimental status, and as such unsuitable as a foundation for our application.

## topics for further consideration

- what should be the proper boundary and interaction between server state and persisted UI state? 

for example in the case of experiment descriptions and tags. when a user is typing it is UI state, then when they submit, it becomes server state. but that doesnt mean we should refresh the entire list of experiments every time a user adds a description. we can get by with actually going in and updating the experiment in the table. but there is a subtle race condition that could arise there with the current approach. since we are doing polling, there may already be a `fetchExperiments` response on its way back when the user submits the modification. so if we update the UI on submit, then update on the stale fetch, we would erroneously revert the update the user made.

maybe that is rare enough that we don't need to worry about it, but it is something to keep in mind. there have been issues caused by stale API response cloberring valid state in the webui before.

that said, one possible solution for the issue would be to replace polling with streaming. already the the polling we do amounts to a pretty significant amount of waste. and if we did streaming, we could simply merge the updates from the server with the updates from the client.

in order for this to work, we would want to have a robust comprehensive approach to how we stream data from the backend. there is some interest on the part of backend team in doing more streaming though. one particular consideration is, since we might want to wait for updates from more than 6 "things" at a time (6 being the max concurrent network requests), we would want to look into either having an streaming API that multiple heterogeneous "things" can share, or doing some kind of connection multiplexing.



