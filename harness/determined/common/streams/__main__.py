from determined.common import streams

from determined.experimental import client


if __name__ == "__main__":
    spec = streams.SubscriptionSpecSet(
        trials=streams.TrialSubscriptionSpec(trial_ids=[1]),
    )
    client.login()
    session = client._determined._session
    s = streams.Stream(session, spec)
    print("event 1", next(s))
    print("event 2", next(s))
    print("event 3", next(s))
    add = streams.SubscriptionSpecSet(
        trials=streams.TrialSubscriptionSpec(trial_ids=[11]),
    )
    print("resubscribing")
    s.resubscribe(add=add, drop=spec)
    for event in s:
        print(event)
