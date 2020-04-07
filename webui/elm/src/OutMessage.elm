module OutMessage exposing
    ( evaluate, evaluateMaybe, evaluateList, evaluateResult
    , mapComponent, mapCmd, mapOutMsg
    , addOutMsg, toNested, fromNested
    , wrap, run
    )

{-| **Note: ** This library is opinionated. The usage of an OutMsg is a technique to extend The Elm Architecture (TEA) to support
child-parent communication. The [README](https://github.com/folkertdev/outmessage/blob/master/README.md) covers the design.

The OutMsg pattern has two components:

  - OutMsg, a user-defined type (just like Model or Msg) with the specific purpose of notifying a parent component.
  - `interpretOutMsg`, a function that converts OutMsg values into side-effects (commands and changes to the model)

OutMsg values can be captured in the parent's update function, and handled there by `interpretOutMsg`.
The basic pattern can be extended to return multiple OutMsg using List or to optionally return no OutMsg using Maybe.

#Evaluators

@docs evaluate, evaluateMaybe, evaluateList, evaluateResult

#Mapping

@docs mapComponent, mapCmd, mapOutMsg

#Helpers

@docs addOutMsg, toNested, fromNested

#Internals

Internal functions that can be used to create custom evaluators.

An evaluator has three basic components:

  - **A state creator**, often using `OutMessage.wrap`.
  - **A state modifier**, any function from the State package (see the use of State.traverse in evaluateList).
  - **A state evaluator** that runs the state and creates a 'vanilla' elm value.

This package uses the [State](http://package.elm-lang.org/packages/folkertdev/elm-state/1.0.0/) package for threading the model through a series of
updates and accumulating commands.

@docs wrap, run

-}

import State exposing (State, state)


swap : ( a, b ) -> ( b, a )
swap ( x, y ) =
    ( y, x )


applyWithDefault : b -> (a -> b) -> Maybe a -> b
applyWithDefault default f =
    Maybe.withDefault default << Maybe.map f


{-| Turn an `OutMsg` value into commands and model changes.

The arguments are:

  - `interpretOutMsg`, a user-defined function that turns OutMsg values into
    model changes and effects.
  - a tuple containing the model (updated with the child component),
    commands (of the parent's Msg type) and an OutMsg. This package exposes
    helpers to construct this tuple from the value that a child update function returns.

Example usage:

    -- in update : Msg -> Model -> (Model, Cmd Msg)
    -- assuming interpretOutMsg : OutMsg -> Model -> (Model, Cmd Msg)
    -- ChildComponentModule.update
    --       : ChildMsg
    --       -> ChildModel -> (ChildModel, Cmd ChildMsg, OutMsg)
    ChildComponentMessageWrapper childMsg ->
        ChildComponentModule.update childMsg model.child
            -- update the model with the new child component
            |> OutMessage.mapComponent
                (\newChild -> { model | child = newChild }
            -- convert child cmd to parent cmd
            |> OutMessage.mapCmd ChildComponentMessageWrapper
            -- apply outmsg changes
            |> OutMessage.evaluate interpretOutMsg

-}
evaluate :
    (outMsg -> model -> ( model, Cmd msg ))
    -> ( model, Cmd msg, outMsg )
    -> ( model, Cmd msg )
evaluate interpretOutMsg ( model, cmd, outMsg ) =
    wrap interpretOutMsg outMsg
        |> run cmd model


{-| Turn a `Maybe OutMsg` into effects and model changes.

Has a third argument for a default command that is used when OutMsg is Nothing.

-}
evaluateMaybe :
    (outMsg -> model -> ( model, Cmd msg ))
    -> Cmd msg
    -> ( model, Cmd msg, Maybe outMsg )
    -> ( model, Cmd msg )
evaluateMaybe interpretOutMsg default ( model, cmd, outMsg ) =
    applyWithDefault (state default) (wrap interpretOutMsg) outMsg
        |> run cmd model


{-| Turn a `Result error OutMsg` into effects and model changes

Has a third argument for a function that turns errors into a command that is used when
OutMsg is Err error.

-}
evaluateResult :
    (outMsg -> model -> ( model, Cmd msg ))
    -> (error -> Cmd msg)
    -> ( model, Cmd msg, Result error outMsg )
    -> ( model, Cmd msg )
evaluateResult interpretOutMsg onErr ( model, cmd, outMsg ) =
    let
        stateful =
            case outMsg of
                Ok v ->
                    wrap interpretOutMsg v

                Err err ->
                    state (onErr err)
    in
    stateful
        |> run cmd model


{-| Turn a `List OutMsg` into effects and model changes.

Takes care of threading the state. When interpreting an OutMsg changes the model,
the updated model will be used for subsequent interpretations of OutMsgs. Cmds are
accumulated and batched.

-}
evaluateList :
    (outMsg -> model -> ( model, Cmd msg ))
    -> ( model, Cmd msg, List outMsg )
    -> ( model, Cmd msg )
evaluateList interpretOutMsg ( model, cmd, outMsgs ) =
    State.traverse (wrap interpretOutMsg) outMsgs
        |> State.map Cmd.batch
        |> run cmd model


{-| Apply a function over the Msg from the child.
-}
mapCmd : (childmsg -> parentmsg) -> ( a, Cmd childmsg, c ) -> ( a, Cmd parentmsg, c )
mapCmd f ( x, cmd, z ) =
    ( x, Cmd.map f cmd, z )


{-| Apply a function over the updated child component.
-}
mapComponent : (childComponent -> a) -> ( childComponent, b, c ) -> ( a, b, c )
mapComponent f ( childComponent, y, z ) =
    ( f childComponent, y, z )


{-| Apply a function over the child's OutMsg.
-}
mapOutMsg : (outMsg -> c) -> ( a, b, outMsg ) -> ( a, b, c )
mapOutMsg f ( x, y, outMsg ) =
    ( x, y, f outMsg )



-- Handy functions


{-| Add an outmessage to the normal type that `update` has. Handy to use in a pipe:

    ( { model | a = 1 }, Cmd.none )
        |> addOutMsg Nothing

-}
addOutMsg : outMsg -> ( model, Cmd msg ) -> ( model, Cmd msg, outMsg )
addOutMsg outMsg ( model, cmd ) =
    ( model, cmd, outMsg )


{-| Helper to split the OutMsg from the normal type that `update` has.

The functions `fst` and `snd` can now be used, which can be handy.

-}
toNested : ( a, b, c ) -> ( ( a, b ), c )
toNested ( x, y, z ) =
    ( ( x, y ), z )


{-| Join the component, command and outmessage into a flat tuple.
-}
fromNested : ( ( a, b ), c ) -> ( a, b, c )
fromNested ( ( x, y ), z ) =
    ( x, y, z )



-- Internals


{-| Embed a function into [State](http://package.elm-lang.org/packages/folkertdev/elm-state/1.0.0/)
-}
wrap : (outmsg -> model -> ( model, Cmd msg )) -> outmsg -> State model (Cmd msg)
wrap f msg =
    State.advance (swap << f msg)


{-| Evaluate a `State model (Cmd msg)` given a model, and commands to prepend.

    wrap interpretOutMsg myOutMsg
        |> run Cmd.none myModel

-}
run : Cmd msg -> model -> State model (Cmd msg) -> ( model, Cmd msg )
run cmd model =
    -- Prepend the child component's Cmds
    State.map (\outCmd -> Cmd.batch [ cmd, outCmd ])
        >> State.run model
        >> swap
