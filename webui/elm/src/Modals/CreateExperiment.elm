module Modals.CreateExperiment exposing
    ( Model
    , Msg
    , OutMsg(..)
    , init
    , openForContinue
    , openForFork
    , subscriptions
    , update
    , view
    )

import API
import APIQL
import Browser.Dom as Dom
import Browser.Events
import Communication as Comm
import Components.YamlEditor as YamlEditor
import Dict exposing (Dict)
import Html as H
import Html.Attributes as HA
import Html.Events as HE
import Json.Decode as D
import Json.Encode as E
import Maybe.Extra
import Page.Common exposing (centeredLoadingWidget)
import Route
import Set
import Task
import Types
import View.Modal as Modal
import Yaml.Encode as YE


{-| The model for the modal when it is open.
-}
type alias OpenModel =
    { editorState : YamlEditor.State
    , content : String
    , badSyntax : Bool
    , parentExperiment : Types.Experiment
    , error : Maybe String
    }


type alias LoadingModel =
    { trialDetail : Types.TrialDetail
    , error : Maybe String
    }


type alias FormErrors =
    { maxSteps : Maybe String
    , description : Maybe String
    }


type alias FormModel =
    { maxSteps : Maybe Int
    , description : String
    , errors : FormErrors
    }


{-| A validated form record.

  - `maxSteps` should be positive.
  - `description` should be nonempty.

-}
type alias ValidatedForm =
    { maxSteps : Int
    , description : String
    }


type Model
    = OpenFork OpenModel
    | OpenContinue OpenModel Types.TrialDetail
    | OpenForm FormModel Types.Experiment Types.TrialDetail
    | Loading LoadingModel
    | Closed


type FormMsg
    = CreateExperimentFromForm
    | EditFullConfig
    | NoOp
    | UpdateFormDescription String
    | UpdateFormMaxSteps String


type Msg
    = GotFormMsg FormMsg
    | CloseModal
    | CreateExperiment
    | ExperimentCreated Types.ExperimentDescriptor
    | NewConfiguration YamlEditor.ContentUpdate
    | GotExperiment Types.ExperimentResult
      -- Errors.
    | GotCriticalError Comm.SystemError
    | GotAPIError API.APIError


type OutMsg
    = CreatedExperiment Types.ID


{-| An ID for the max steps input so we can focus it in JS.
-}
inputMaxStepsId : String
inputMaxStepsId =
    "input-max-steps"


requestHandlers : (body -> Msg) -> API.RequestHandlers Msg body
requestHandlers onSuccess =
    { onSuccess = onSuccess
    , onSystemError = GotCriticalError
    , onAPIError = GotAPIError
    }


editorConfig : YamlEditor.Config Msg
editorConfig =
    { newContentToMsg = NewConfiguration
    , containerIdOverride = Nothing
    }


init : Model
init =
    Closed


updateOpenFork : Msg -> OpenModel -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
updateOpenFork msg openModel =
    let
        nullAction =
            ( OpenFork openModel, Cmd.none, Nothing )
    in
    case msg of
        GotFormMsg _ ->
            nullAction

        CloseModal ->
            ( Closed
            , YamlEditor.destroy openModel.editorState
            , Nothing
            )

        NewConfiguration c ->
            let
                subModel =
                    { openModel
                        | content = c.content
                        , badSyntax = c.badSyntax
                    }
            in
            ( OpenFork subModel
            , YamlEditor.resize openModel.editorState
            , Nothing
            )

        CreateExperiment ->
            ( OpenFork openModel, doCreateExperiment openModel, Nothing )

        ExperimentCreated descriptor ->
            ( OpenFork openModel
            , YamlEditor.destroy openModel.editorState
            , Comm.OutMessage (CreatedExperiment descriptor.id) |> Just
            )

        GotExperiment _ ->
            nullAction

        GotCriticalError e ->
            ( OpenFork openModel, Cmd.none, Comm.Error e |> Just )

        GotAPIError _ ->
            let
                errorStr =
                    "An unknown error occurred.  Ensure that the experiment configuration is correct. "
                        ++ "If the problem persists, please contact us."

                newModel =
                    { openModel
                        | error = Just errorStr
                    }
            in
            ( OpenFork newModel
            , YamlEditor.resize openModel.editorState
            , Nothing
            )


updateOpenContinue : Msg -> OpenModel -> Types.TrialDetail -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
updateOpenContinue msg openModel trialDetail =
    let
        nullAction =
            ( OpenContinue openModel trialDetail, Cmd.none, Nothing )
    in
    case msg of
        GotFormMsg _ ->
            nullAction

        CloseModal ->
            ( Closed
            , YamlEditor.destroy openModel.editorState
            , Nothing
            )

        NewConfiguration c ->
            let
                subModel =
                    { openModel
                        | content = c.content
                        , badSyntax = c.badSyntax
                    }
            in
            ( OpenContinue subModel trialDetail, Cmd.none, Nothing )

        CreateExperiment ->
            ( OpenContinue openModel trialDetail, doCreateExperiment openModel, Nothing )

        ExperimentCreated descriptor ->
            ( OpenContinue openModel trialDetail
            , YamlEditor.destroy openModel.editorState
            , Comm.OutMessage (CreatedExperiment descriptor.id) |> Just
            )

        GotExperiment _ ->
            nullAction

        GotCriticalError e ->
            ( OpenContinue openModel trialDetail, Cmd.none, Comm.Error e |> Just )

        GotAPIError _ ->
            let
                errorStr =
                    "An unknown error occurred.  Ensure that the experiment configuration is correct. "
                        ++ "If the problem persists, please contact us."

                newModel =
                    { openModel
                        | error = Just errorStr
                    }
            in
            ( OpenContinue newModel trialDetail
            , YamlEditor.resize openModel.editorState
            , Nothing
            )


updateOpenForm : Msg -> FormModel -> Types.Experiment -> Types.TrialDetail -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
updateOpenForm msg formModel experiment trialDetail =
    let
        nullAction =
            ( OpenForm formModel experiment trialDetail, Cmd.none, Nothing )
    in
    case msg of
        GotFormMsg formMsg ->
            case formMsg of
                CreateExperimentFromForm ->
                    case validateForm formModel of
                        Ok validatedForm ->
                            let
                                configStr =
                                    getConfigAsYamlString experiment (Just ( trialDetail, validatedForm ))
                            in
                            ( OpenForm formModel experiment trialDetail
                            , API.createExperiment
                                (requestHandlers ExperimentCreated)
                                experiment.id
                                configStr
                            , Nothing
                            )

                        Err formErrors ->
                            let
                                newFormModel =
                                    { formModel | errors = formErrors }
                            in
                            ( OpenForm newFormModel experiment trialDetail, Cmd.none, Nothing )

                EditFullConfig ->
                    case validateForm formModel of
                        Ok validatedForm ->
                            let
                                ( newModel, cmd ) =
                                    toOpenState experiment (Just ( trialDetail, validatedForm ))
                            in
                            ( newModel, cmd, Nothing )

                        Err formErrors ->
                            let
                                newFormModel =
                                    { formModel | errors = formErrors }
                            in
                            ( OpenForm newFormModel experiment trialDetail, Cmd.none, Nothing )

                NoOp ->
                    nullAction

                UpdateFormDescription description ->
                    let
                        newFormModel =
                            { formModel | description = description }
                    in
                    ( OpenForm newFormModel experiment trialDetail, Cmd.none, Nothing )

                UpdateFormMaxSteps maxSteps ->
                    let
                        newFormModel =
                            { formModel | maxSteps = String.toInt maxSteps }
                    in
                    ( OpenForm newFormModel experiment trialDetail, Cmd.none, Nothing )

        CloseModal ->
            ( Closed, Cmd.none, Nothing )

        ExperimentCreated descriptor ->
            ( OpenForm formModel experiment trialDetail
            , Cmd.none
            , Comm.OutMessage (CreatedExperiment descriptor.id) |> Just
            )

        _ ->
            nullAction


updateLoading : Msg -> LoadingModel -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
updateLoading msg loadingModel =
    let
        nullAction =
            ( Loading loadingModel, Cmd.none, Nothing )

        errorStr =
            "An error occurred while loading the experiment configuration.  "
                ++ "If the error persists, please contact us."

        errorModel =
            { loadingModel | error = Just errorStr }
    in
    case msg of
        GotFormMsg _ ->
            nullAction

        CloseModal ->
            ( Closed, Cmd.none, Nothing )

        NewConfiguration _ ->
            nullAction

        CreateExperiment ->
            nullAction

        ExperimentCreated _ ->
            nullAction

        GotExperiment (Ok experiment) ->
            let
                configDescription =
                    getDescriptionFromExperimentAndTrial experiment
                        (Just loadingModel.trialDetail)

                formModel =
                    { maxSteps = Nothing
                    , description = configDescription
                    , errors = { maxSteps = Nothing, description = Nothing }
                    }
            in
            ( OpenForm formModel experiment loadingModel.trialDetail
            , focusMaxSteps
            , Nothing
            )

        GotExperiment (Err _) ->
            ( Loading errorModel, Cmd.none, Nothing )

        GotCriticalError e ->
            ( Loading loadingModel, Cmd.none, Comm.Error e |> Just )

        GotAPIError _ ->
            ( Loading errorModel, Cmd.none, Nothing )


update : Msg -> Model -> ( Model, Cmd Msg, Maybe (Comm.OutMessage OutMsg) )
update msg model =
    case model of
        OpenFork openModel ->
            updateOpenFork msg openModel

        OpenContinue openModel trialDetail ->
            updateOpenContinue msg openModel trialDetail

        OpenForm formModel experiment trialDetail ->
            updateOpenForm msg formModel experiment trialDetail

        Loading loadingModel ->
            updateLoading msg loadingModel

        Closed ->
            ( Closed, Cmd.none, Nothing )


doCreateExperiment : OpenModel -> Cmd Msg
doCreateExperiment openModel =
    API.createExperiment
        (requestHandlers ExperimentCreated)
        openModel.parentExperiment.id
        openModel.content


openForContinue : Types.TrialDetail -> ( Model, Cmd Msg )
openForContinue trialDetail =
    let
        cmd =
            APIQL.sendQuery
                -- TODO: Get rid of the whole ExperimentResult thing, now that we have better-typed
                -- responses from GraphQL.
                (requestHandlers (GotExperiment << Maybe.Extra.unwrap (Err (Types.ExperimentDescriptor 0 False "" Set.empty)) Ok))
                (APIQL.experimentDetailQuery trialDetail.experimentId)

        loadingModel =
            { trialDetail = trialDetail
            , error = Nothing
            }
    in
    ( Loading loadingModel, cmd )


openForFork : Types.Experiment -> ( Model, Cmd Msg )
openForFork experiment =
    toOpenState experiment Nothing


toOpenState : Types.Experiment -> Maybe ( Types.TrialDetail, ValidatedForm ) -> ( Model, Cmd Msg )
toOpenState experiment maybeTrialAndValidatedForm =
    let
        ( editorState, cmd ) =
            YamlEditor.init editorConfig (getConfigAsYamlString experiment maybeTrialAndValidatedForm)

        state =
            { editorState = editorState

            -- The content will be updated by the editor as soon as it is set up.
            , content = ""
            , badSyntax = False
            , parentExperiment = experiment
            , error = Nothing
            }
    in
    case maybeTrialAndValidatedForm of
        Just ( trialDetail, _ ) ->
            ( OpenContinue state trialDetail, cmd )

        Nothing ->
            ( OpenFork state, cmd )


validateForm : FormModel -> Result FormErrors ValidatedForm
validateForm formModel =
    let
        maxSteps =
            Maybe.withDefault 0 formModel.maxSteps

        maxStepsError =
            if maxSteps <= 0 then
                Just "Provide a valid number of steps that is larger than 0."

            else
                Nothing

        descriptionError =
            if String.isEmpty formModel.description then
                Just "Provide an experiment description."

            else
                Nothing
    in
    case ( maxStepsError, descriptionError ) of
        ( Nothing, Nothing ) ->
            Ok { maxSteps = maxSteps, description = formModel.description }

        _ ->
            Err { maxSteps = maxStepsError, description = descriptionError }



---- Utility functions for inspecting and modifying experiment configurations.


getConfigAsYamlString : Types.Experiment -> Maybe ( Types.TrialDetail, ValidatedForm ) -> String
getConfigAsYamlString experiment maybeTrialAndValidatedForm =
    let
        updateForTrial =
            case maybeTrialAndValidatedForm of
                Just ( trialDetail, validatedForm ) ->
                    replaceWithConstHParams trialDetail >> replaceWithSingleSearcher trialDetail validatedForm

                Nothing ->
                    identity

        description =
            case maybeTrialAndValidatedForm of
                Just ( _, validatedForm ) ->
                    validatedForm.description

                Nothing ->
                    getDescriptionFromExperimentAndTrial experiment Nothing
    in
    experiment.config
        -- Update fields in the config.
        |> replaceDescription description
        |> updateForTrial
        -- Turn the config back into a single D.Value.
        |> E.dict identity identity
        -- Serialize the config to YAML.
        |> YE.encode


getDescriptionFromConfig : Types.ExperimentConfig -> Maybe String
getDescriptionFromConfig config =
    Dict.get "description" config
        |> Maybe.andThen (Result.toMaybe << D.decodeValue D.string)


getDescriptionFromExperimentAndTrial : Types.Experiment -> Maybe Types.TrialDetail -> String
getDescriptionFromExperimentAndTrial experiment maybeTrialDetail =
    let
        prefix =
            case maybeTrialDetail of
                Just trialDetail ->
                    "Continuation of trial " ++ String.fromInt trialDetail.id ++ ", "

                Nothing ->
                    "Fork of "

        suffix =
            case getDescriptionFromConfig experiment.config of
                Just desc ->
                    " (" ++ desc ++ ")"

                Nothing ->
                    ""
    in
    prefix ++ "experiment " ++ String.fromInt experiment.id ++ suffix


replaceConfigurationValue : String -> E.Value -> Types.ExperimentConfig -> Types.ExperimentConfig
replaceConfigurationValue key value =
    Dict.update key (\_ -> Just value)


replaceDescription : String -> Types.ExperimentConfig -> Types.ExperimentConfig
replaceDescription newDescription =
    replaceConfigurationValue "description" (E.string newDescription)


trialHParamsToExperimentHParams : Dict String E.Value -> E.Value
trialHParamsToExperimentHParams =
    E.dict
        identity
        (\value -> E.object [ ( "type", E.string "const" ), ( "val", value ) ])


replaceWithConstHParams : Types.TrialDetail -> Types.ExperimentConfig -> Types.ExperimentConfig
replaceWithConstHParams trial =
    replaceConfigurationValue "hyperparameters" (trialHParamsToExperimentHParams trial.hparams)


replaceWithSingleSearcher : Types.TrialDetail -> ValidatedForm -> Types.ExperimentConfig -> Types.ExperimentConfig
replaceWithSingleSearcher trial validatedForm =
    let
        createSearcherConfig : Maybe D.Value -> Maybe D.Value
        createSearcherConfig maybeOldConfig =
            let
                maybeOldDict : Maybe (Dict String D.Value)
                maybeOldDict =
                    maybeOldConfig |> Maybe.andThen (D.decodeValue (D.dict D.value) >> Result.toMaybe)

                copySearcherKey key =
                    maybeOldDict
                        |> Maybe.andThen (Dict.get key)
                        |> Maybe.map (\x -> ( key, x ))
            in
            [ copySearcherKey "metric"
            , copySearcherKey "smaller_is_better"
            , Just ( "name", E.string "single" )
            , Just ( "source_trial_id", E.int trial.id )
            , Just ( "max_steps", E.int validatedForm.maxSteps )
            ]
                |> Maybe.Extra.values
                |> E.object
                |> Just
    in
    Dict.update "searcher" createSearcherConfig



---- View.


renderModal : H.Html Msg -> H.Html Msg
renderModal c =
    Modal.view
        { content = c
        , attributes = [ HA.style "width" "50vw", HA.style "height" "70vh" ]
        , closeMsg = CloseModal
        }


renderHeader : Types.Experiment -> Maybe Types.TrialDetail -> H.Html Msg
renderHeader experiment maybeTrialDetail =
    let
        expIdStr =
            String.fromInt experiment.id

        title =
            case maybeTrialDetail of
                Just trialDetail ->
                    "Continue Trial " ++ String.fromInt trialDetail.id ++ " of Experiment " ++ expIdStr

                Nothing ->
                    "Fork Experiment " ++ expIdStr

        experimentUrl =
            Route.toString (Route.ExperimentDetail experiment.id)

        experimentLink =
            H.a
                [ HA.href experimentUrl
                , HA.target "blank_"
                , HA.class "text-blue-800 font-medium hover:underline"
                ]
                [ H.text ("Experiment " ++ expIdStr) ]

        titleClasses =
            "pt-2 " ++ Modal.titleClasses

        descriptionClasses =
            "mt-2"
    in
    H.div []
        [ H.div [ HA.class titleClasses ] [ H.text title ]
        , H.div [ HA.class descriptionClasses ]
            [ H.text "Copied model definition from "
            , experimentLink
            ]
        ]


renderFormError : Maybe String -> H.Html FormMsg
renderFormError error =
    case error of
        Just errorMessage ->
            H.div [ HA.class "mt-2 text-red-500" ] [ H.text errorMessage ]

        Nothing ->
            H.text ""


viewOpen : H.Html Msg -> OpenModel -> H.Html Msg
viewOpen header openModel =
    let
        maybeError =
            case openModel.error of
                Just msg ->
                    H.div
                        [ HA.class "text-red-500 font-semibold"
                        , HA.style "max-width" "80%"
                        ]
                        [ H.text msg ]

                Nothing ->
                    H.text ""

        body =
            H.div
                [ HA.class "border rounded flex-grow relative", HA.style "height" "100%" ]
                [ YamlEditor.view openModel.editorState ]

        footer =
            H.div []
                [ H.div [ HA.class "flex flex-row justify-center" ] [ maybeError ]
                , H.div [ HA.class "flex flex-row justify-end" ]
                    [ badSyntaxMessage openModel.badSyntax
                    ]
                ]

        content =
            Modal.contentView
                { header = header
                , body = body
                , footer = Just footer
                , buttons = [ createButton (not openModel.badSyntax) ]
                }
    in
    renderModal content


viewOpenFork : OpenModel -> H.Html Msg
viewOpenFork openModel =
    let
        header =
            renderHeader openModel.parentExperiment Nothing
    in
    viewOpen header openModel


viewOpenContinue : OpenModel -> Types.TrialDetail -> H.Html Msg
viewOpenContinue openModel trialDetail =
    let
        header =
            renderHeader openModel.parentExperiment (Just trialDetail)
    in
    viewOpen header openModel


viewForm : FormModel -> Types.Experiment -> Types.TrialDetail -> H.Html Msg
viewForm formModel experiment trialDetail =
    let
        header =
            renderHeader experiment (Just trialDetail)

        formMaxSteps =
            Maybe.Extra.unwrap "" String.fromInt formModel.maxSteps

        formErrorMaxSteps =
            renderFormError formModel.errors.maxSteps

        formErrorDescription =
            renderFormError formModel.errors.description

        buttonEditFullConfig =
            Page.Common.buttonCreator
                { action = Page.Common.SendMsg EditFullConfig
                , bgColor = "blue"
                , fgColor = "white"
                , isActive = True
                , isPending = False
                , style = Page.Common.TextOnly
                , text = "Edit Full Config"
                }
                |> H.map GotFormMsg

        buttonCreate =
            Page.Common.buttonCreator
                { action = Page.Common.SendMsg CreateExperimentFromForm
                , bgColor = "blue"
                , fgColor = "white"
                , isActive = True
                , isPending = False
                , style = Page.Common.TextOnly
                , text = "Create"
                }
                |> H.map GotFormMsg

        body =
            H.div []
                [ H.div [ HA.class "flex-grow" ]
                    [ H.label [ HA.class "block text-gray-700 text-sm font-bold mb-2" ]
                        [ H.text "Number of steps to train for" ]
                    , H.input
                        [ HE.onInput UpdateFormMaxSteps
                        , HA.id inputMaxStepsId
                        , HA.class "appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
                        , HA.type_ "number"
                        , HA.min "0"
                        , HA.value formMaxSteps
                        ]
                        []
                    , formErrorMaxSteps
                    , H.label [ HA.class "block text-gray-700 text-sm font-bold mt-4 mb-2" ]
                        [ H.text "Experiment description" ]
                    , H.input
                        [ HE.onInput UpdateFormDescription
                        , HA.class "appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
                        , HA.value formModel.description
                        ]
                        []
                    , formErrorDescription
                    ]
                ]
                |> H.map GotFormMsg
    in
    Modal.contentView
        { header = header
        , body = body
        , footer = Nothing
        , buttons = [ buttonEditFullConfig, buttonCreate ]
        }
        |> renderModal


viewLoading : LoadingModel -> H.Html Msg
viewLoading loadingModel =
    let
        inner =
            case loadingModel.error of
                Just msg ->
                    H.div [ HA.class "text-red-500 text-2xl" ]
                        [ H.text msg ]

                Nothing ->
                    centeredLoadingWidget
    in
    Modal.contentView
        { header = H.text "Loading"
        , body = inner
        , footer = Nothing
        , buttons = []
        }
        |> renderModal


view : Model -> H.Html Msg
view model =
    case model of
        OpenFork openModel ->
            viewOpenFork openModel

        OpenContinue openModel trialDetail ->
            viewOpenContinue openModel trialDetail

        OpenForm formModel experiment trialDetail ->
            viewForm formModel experiment trialDetail

        Loading loadingModel ->
            viewLoading loadingModel

        Closed ->
            H.text ""


badSyntaxMessage : Bool -> H.Html Msg
badSyntaxMessage enabled =
    if enabled then
        H.div [ HA.class "text-red-500 text-lg mr-4 font-bold" ]
            [ H.text "Bad YAML syntax" ]

    else
        H.text ""


createButton : Bool -> H.Html Msg
createButton enabled =
    Page.Common.buttonCreator
        { action = Page.Common.SendMsg CreateExperiment
        , bgColor = "blue"
        , fgColor = "white"
        , isActive = enabled
        , isPending = False
        , style = Page.Common.TextOnly
        , text = "Create"
        }


subscriptions : Model -> Sub Msg
subscriptions model =
    let
        escapeDecoder =
            D.field "key" D.string
                |> D.andThen
                    (\key ->
                        if key == "Escape" then
                            D.succeed CloseModal

                        else
                            D.fail "not Escape"
                    )
    in
    Sub.batch
        [ case model of
            OpenFork _ ->
                YamlEditor.subscriptions editorConfig

            OpenContinue _ _ ->
                YamlEditor.subscriptions editorConfig

            OpenForm _ _ _ ->
                Sub.none

            Loading _ ->
                Sub.none

            Closed ->
                Sub.none
        , Browser.Events.onKeyUp escapeDecoder
        ]


focusMaxSteps : Cmd Msg
focusMaxSteps =
    Task.attempt (always (GotFormMsg NoOp)) (Dom.focus inputMaxStepsId)
