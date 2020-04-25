module Main exposing (main)

import API
import Authentication exposing (doLogin, getCurrentUser)
import Browser
import Browser.Dom
import Browser.Navigation as Navigation
import Communication as Comm
import Model exposing (Model, Page(..))
import Msg exposing (Msg(..))
import OutMessage as M
import Page.Cluster
import Page.CommandList
import Page.ExperimentDetail
import Page.ExperimentList
import Page.LogViewer
import Page.Login
import Page.Logout
import Page.NotebookList
import Page.ShellList
import Page.TensorBoardList
import Page.TrialDetail
import Ports
import Process
import Route
import Session exposing (Session)
import Task
import Time
import Toast
import Types
import Url
import View
import View.SlotChart


main : Program String Model Msg
main =
    Browser.application
        { init = init
        , view = view
        , update = update
        , onUrlRequest = UrlRequested
        , onUrlChange = UrlChanged
        , subscriptions = subscriptions
        }


mapInit : Model -> PageInfo model msg outMsg -> ( model, Cmd msg ) -> ( Model, Cmd Msg )
mapInit model info ( pageModel, pageCmd ) =
    ( pageModel, Cmd.batch [ Ports.kickResizePort (), pageCmd ], Nothing )
        |> makePageMapper info model


init : String -> Url.Url -> Navigation.Key -> ( Model, Cmd Msg )
init version url key =
    let
        initial =
            { session =
                { key = key
                , zone = Time.utc
                , user = Nothing
                }
            , info = Nothing
            , page = Init
            , criticalError = Nothing
            , toasts = []
            , nextToastID = 0
            , slots = Nothing
            , slotsRequestPending = True
            , userDropdownOpen = False
            , previousExperimentListModel = Nothing
            , previousCommandListModel = Nothing
            , previousNotebookListModel = Nothing
            , previousShellListModel = Nothing
            , previousTensorBoardListModel = Nothing
            , version = version
            }
    in
    ( initial
    , Cmd.batch
        [ getCurrentUser (ValidatedAuthentication False url)
        , API.fetchDeterminedInfo (requestHandlers GotDeterminedInfo)
        , API.pollSlots (requestHandlers GotSlots)
        , Task.perform GotTimeZone Time.here

        -- Dirty hack: focus the big scrolling div shortly after page load so that keyboard
        -- scrolling works without the user having to manually focus it. For some reason, running
        -- the task without a sleep (or with a sleep of insufficient duration) reports success but
        -- doesn't have the desired effect.
        , Task.attempt (always NoOp)
            (Process.sleep 500 |> Task.andThen (always (Browser.Dom.focus "det-main-container")))
        ]
    )


requestHandlers : (body -> Msg) -> API.RequestHandlers Msg body
requestHandlers onSuccess =
    { onSuccess = onSuccess
    , onSystemError = GotCriticalError
    , onAPIError = GotAPIError
    }


view : Model -> Browser.Document Msg
view model =
    { title = "Determined Deep Learning Training Platform"
    , body = View.viewBody model
    }


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    let
        pageUpdate info pageMsg pageModel =
            info.update pageMsg pageModel model.session
                |> makePageMapper info model
    in
    case msg of
        NoOp ->
            ( model, Cmd.none )

        GotDeterminedInfo info ->
            let
                cmds =
                    case ( info.telemetry.enabled, info.telemetry.segmentKey ) of
                        ( True, Just segmentKey ) ->
                            Cmd.batch
                                [ Ports.loadAnalytics segmentKey
                                , Ports.setAnalyticsIdentityPort info.clusterId
                                ]

                        _ ->
                            Cmd.none
            in
            ( { model | info = Just info }, cmds )

        GotTimeZone zone ->
            let
                session =
                    model.session
            in
            ( { model | session = { session | zone = zone } }, Cmd.none )

        UrlRequested req ->
            case req of
                Browser.Internal url ->
                    let
                        loadOrReload options defaultOptions initFn previousModel pageInfo =
                            if options == defaultOptions then
                                initFn previousModel options
                                    |> mapInit model pageInfo

                            else
                                ( model, Navigation.pushUrl model.session.key (Url.toString url) )
                    in
                    case ( model.page, Route.parse url ) of
                        -- There are URL requests (e.g., the link to a model def download) that come
                        -- through as "internal" (i.e., on this domain) but should not be handled in
                        -- Elm.
                        ( _, Nothing ) ->
                            ( model, Navigation.load (Url.toString url) )

                        -- If we are on the experiment list page, a request for the default
                        -- experiment list page should refresh the page without any change to the
                        -- URL. Otherwise, go ahead and change the URL.
                        ( ExperimentList _, Just (Route.ExperimentList options) ) ->
                            loadOrReload
                                options
                                Route.defaultExperimentListOptions
                                Page.ExperimentList.init
                                model.previousExperimentListModel
                                experimentListInfo

                        ( CommandList _, Just (Route.CommandList options) ) ->
                            loadOrReload
                                options
                                Route.defaultCommandLikeListOptions
                                Page.CommandList.init
                                model.previousCommandListModel
                                commandListInfo

                        ( NotebookList _, Just (Route.NotebookList options) ) ->
                            loadOrReload
                                options
                                Route.defaultCommandLikeListOptions
                                Page.NotebookList.init
                                model.previousNotebookListModel
                                notebookListInfo

                        ( ShellList _, Just (Route.ShellList options) ) ->
                            loadOrReload
                                options
                                Route.defaultCommandLikeListOptions
                                Page.ShellList.init
                                model.previousShellListModel
                                shellListInfo

                        ( TensorBoardList _, Just (Route.TensorBoardList options) ) ->
                            loadOrReload
                                options
                                Route.defaultCommandLikeListOptions
                                Page.TensorBoardList.init
                                model.previousTensorBoardListModel
                                tensorboardListInfo

                        _ ->
                            let
                                navCmd =
                                    Navigation.pushUrl model.session.key (Url.toString url)

                                cmds =
                                    case model.info of
                                        Just info ->
                                            if info.telemetry.enabled then
                                                Cmd.batch [ Ports.setAnalyticsPagePort url.path, navCmd ]

                                            else
                                                navCmd

                                        _ ->
                                            navCmd
                            in
                            ( model, cmds )

                Browser.External href ->
                    ( model
                    , Navigation.load href
                    )

        UrlChanged url ->
            updateWithRoute url model

        ToastExpired id ->
            let
                newToasts =
                    model.toasts
                        |> List.filter (\t -> t.id /= id)
            in
            ( { model | toasts = newToasts }, Cmd.none )

        SlotsTick ->
            if model.slotsRequestPending then
                ( model, Cmd.none )

            else
                ( { model | slotsRequestPending = True }
                , API.pollSlots (requestHandlers GotSlots)
                )

        GotSlots slots ->
            let
                isActive =
                    View.SlotChart.busySlots slots
                        |> List.length
                        |> (/=) 0

                faviconSuffix =
                    if isActive then
                        "-active"

                    else
                        ""

                updateFaviconCmd =
                    Ports.setFavicon ("/favicons/favicon" ++ faviconSuffix ++ ".png")
            in
            ( { model | slots = Just slots, slotsRequestPending = False }
            , updateFaviconCmd
            )

        ToggleUserDropdownMenu value ->
            ( { model | userDropdownOpen = value }, Cmd.none )

        -- Authentication stuff.
        ValidatedAuthentication autoLogin url result ->
            case result of
                Ok user ->
                    updateWithRoute url (setSessionUser model (Just user))

                Err _ ->
                    if not autoLogin then
                        let
                            credentials =
                                { username = "determined"
                                , password = ""
                                }
                        in
                        ( model, doLogin (GotAuthenticationResponse url) credentials )

                    else
                        let
                            newUrl =
                                Route.toString (Route.Login (Just (Url.toString url)))
                        in
                        ( model, Navigation.pushUrl model.session.key newUrl )

        GotAuthenticationResponse url result ->
            case result of
                Ok _ ->
                    ( model, getCurrentUser (ValidatedAuthentication True url) )

                Err _ ->
                    let
                        newUrl =
                            Route.toString (Route.Login (Just (Url.toString url)))
                    in
                    ( model, Navigation.pushUrl model.session.key newUrl )

        -- Individual pages.
        ClusterMsg pageMsg ->
            case model.page of
                Cluster pageModel ->
                    pageUpdate clusterInfo pageMsg pageModel

                _ ->
                    ( model, Cmd.none )

        CommandListMsg pageMsg ->
            case model.page of
                CommandList pageModel ->
                    pageUpdate commandListInfo pageMsg pageModel

                _ ->
                    ( model, Cmd.none )

        ExperimentDetailMsg pageMsg ->
            case model.page of
                ExperimentDetail pageModel ->
                    pageUpdate experimentDetailInfo pageMsg pageModel

                _ ->
                    ( model, Cmd.none )

        ExperimentListMsg pageMsg ->
            case model.page of
                ExperimentList pageModel ->
                    pageUpdate experimentListInfo pageMsg pageModel

                _ ->
                    ( model, Cmd.none )

        LoginMsg pageMsg ->
            case model.page of
                Login pageModel ->
                    pageUpdate loginInfo pageMsg pageModel

                _ ->
                    ( model, Cmd.none )

        LogoutMsg pageMsg ->
            case model.page of
                Logout pageModel ->
                    pageUpdate logoutInfo pageMsg pageModel

                _ ->
                    ( model, Cmd.none )

        NotebookListMsg pageMsg ->
            case model.page of
                NotebookList pageModel ->
                    pageUpdate notebookListInfo pageMsg pageModel

                _ ->
                    ( model, Cmd.none )

        ShellListMsg pageMsg ->
            case model.page of
                ShellList pageModel ->
                    pageUpdate shellListInfo pageMsg pageModel

                _ ->
                    ( model, Cmd.none )

        TensorBoardListMsg pageMsg ->
            case model.page of
                TensorBoardList pageModel ->
                    pageUpdate tensorboardListInfo pageMsg pageModel

                _ ->
                    ( model, Cmd.none )

        TrialDetailMsg pageMsg ->
            case model.page of
                TrialDetail pageModel ->
                    pageUpdate trialDetailInfo pageMsg pageModel

                _ ->
                    ( model, Cmd.none )

        LogViewerMsg pageMsg ->
            case model.page of
                LogViewer pageModel ->
                    pageUpdate logViewerInfo pageMsg pageModel

                _ ->
                    ( model, Cmd.none )

        -- Errors.
        GotCriticalError error ->
            let
                ( subModel, subCmd ) =
                    processSystemError model error
            in
            -- This error message is only triggered if request to API.pollSlots fails
            -- hence we can safely reset the state of the slots request. If other requests start
            -- using the same handler they would need to distinguish which request has failed to
            -- only reset the slotsRequestPending status only if the corresponding request failed.
            ( { subModel | slotsRequestPending = False }, subCmd )

        GotAPIError error ->
            let
                -- TODO(jgevirtz): Report error to user.
                _ =
                    Debug.log "Got error" error
            in
            -- This error message is only triggered if request to API.pollSlots fails
            -- hence we can safely reset the state of the slots request. If other requests start
            -- using the same handler they would need to distinguish which request has failed to
            -- only reset the slotsRequestPending status only if the corresponding request failed.
            ( { model | slotsRequestPending = False }, Cmd.none )


updateWithRoute : Url.Url -> Model -> ( Model, Cmd Msg )
updateWithRoute url model =
    let
        ( newModel, cmd ) =
            case Route.parse url of
                Just Route.Cluster ->
                    Page.Cluster.init |> mapInit model clusterInfo

                Just (Route.CommandList options) ->
                    case model.page of
                        CommandList _ ->
                            ( model, Cmd.none )

                        _ ->
                            Page.CommandList.init model.previousCommandListModel options
                                |> mapInit model commandListInfo

                Just Route.Dashboard ->
                    ( model, Navigation.load (Route.toString Route.Dashboard) )

                Just (Route.ExperimentDetail id) ->
                    Page.ExperimentDetail.init id |> mapInit model experimentDetailInfo

                Just (Route.ExperimentList options) ->
                    case model.page of
                        ExperimentList _ ->
                            ( model, Cmd.none )

                        _ ->
                            Page.ExperimentList.init model.previousExperimentListModel options
                                |> mapInit model experimentListInfo

                Just (Route.Login _) ->
                    Page.Login.init |> mapInit model loginInfo

                Just Route.Logout ->
                    Page.Logout.init |> mapInit model logoutInfo

                Just (Route.NotebookList options) ->
                    case model.page of
                        NotebookList _ ->
                            ( model, Cmd.none )

                        _ ->
                            Page.NotebookList.init model.previousNotebookListModel options |> mapInit model notebookListInfo

                Just (Route.ShellList options) ->
                    case model.page of
                        ShellList _ ->
                            ( model, Cmd.none )

                        _ ->
                            Page.ShellList.init model.previousShellListModel options
                                |> mapInit model shellListInfo

                Just (Route.TensorBoardList options) ->
                    case model.page of
                        TensorBoardList _ ->
                            ( model, Cmd.none )

                        _ ->
                            Page.TensorBoardList.init model.previousTensorBoardListModel options
                                |> mapInit model tensorboardListInfo

                Just (Route.TrialDetail id) ->
                    Page.TrialDetail.init id |> mapInit model trialDetailInfo

                Just (Route.LogViewer id) ->
                    Page.LogViewer.init id |> mapInit model logViewerInfo

                Nothing ->
                    ( { model | page = NotFound }, Cmd.none )
    in
    case model.page of
        ExperimentList m ->
            ( { newModel | previousExperimentListModel = Just m }, cmd )

        CommandList m ->
            ( { newModel | previousCommandListModel = Just m }, cmd )

        NotebookList m ->
            ( { newModel | previousNotebookListModel = Just m }, cmd )

        ShellList m ->
            ( { newModel | previousShellListModel = Just m }, cmd )

        TensorBoardList m ->
            ( { newModel | previousTensorBoardListModel = Just m }, cmd )

        _ ->
            ( newModel, cmd )


setSessionUser : Model -> Maybe Types.SessionUser -> Model
setSessionUser model user =
    let
        oldSession =
            model.session

        newSession =
            { oldSession | user = user }

        newModel =
            { model | session = newSession }
    in
    newModel


processSystemError : Model -> Comm.SystemError -> ( Model, Cmd Msg )
processSystemError m e =
    case e of
        Comm.AuthenticationError ->
            let
                url =
                    Route.toString (Route.Login Nothing)

                cmd =
                    Navigation.pushUrl m.session.key url

                oldSession =
                    m.session

                newSession =
                    { oldSession | user = Nothing }
            in
            ( { m | session = newSession }, cmd )

        _ ->
            -- TODO(jgevirtz): Notify users of errors.
            ( m, Cmd.none )



---- Subscriptions.


appSubscriptions : Model -> Sub Msg
appSubscriptions model =
    -- When no user is logged in, polling slots will fail. When the cluster page is up, it handles
    -- polling slots and sending the results back out.
    case ( model.session.user, model.page ) of
        ( Nothing, _ ) ->
            Sub.none

        ( _, Cluster _ ) ->
            Sub.none

        _ ->
            Time.every 2000 (always SlotsTick)


subscriptions : Model -> Sub Msg
subscriptions model =
    let
        pageSubs info pageModel =
            info.subscriptions pageModel |> Sub.map info.msgTagger

        childSubs =
            case model.page of
                Init ->
                    Sub.none

                NotFound ->
                    Sub.none

                Cluster childModel ->
                    pageSubs clusterInfo childModel

                CommandList childModel ->
                    pageSubs commandListInfo childModel

                ExperimentDetail childModel ->
                    pageSubs experimentDetailInfo childModel

                ExperimentList childModel ->
                    pageSubs experimentListInfo childModel

                Login childModel ->
                    pageSubs loginInfo childModel

                Logout childModel ->
                    pageSubs logoutInfo childModel

                NotebookList childModel ->
                    pageSubs notebookListInfo childModel

                ShellList childModel ->
                    pageSubs shellListInfo childModel

                TensorBoardList childModel ->
                    pageSubs tensorboardListInfo childModel

                TrialDetail childModel ->
                    pageSubs trialDetailInfo childModel

                LogViewer childModel ->
                    pageSubs logViewerInfo childModel
    in
    Sub.batch [ appSubscriptions model, childSubs ]



---- Subpage handling setup.


{-| A record that gathers up all of the functions that define the interface of a page. Defining this
type greatly reduces duplication in many places, since it becomes possible to define a single
function that takes a PageInfo and does the desired thing independent of the current page.
-}
type alias PageInfo model msg outMsg =
    { pageTagger : model -> Page
    , msgTagger : msg -> Msg
    , subscriptions : model -> Sub msg
    , update : msg -> model -> Session -> ( model, Cmd msg, Maybe (Comm.OutMessage outMsg) )
    , outHandler : outMsg -> Model -> ( Model, Cmd Msg )
    }


{-| evaluateCommOutMessage handles Maybe (Communication.OutMessage) values coming out of
page update functions. This is the heart of the global error handling system.
-}
evaluateCommOutMessage : (outMsg -> Model -> ( Model, Cmd Msg )) -> ( Model, Cmd Msg, Maybe (Comm.OutMessage outMsg) ) -> ( Model, Cmd Msg )
evaluateCommOutMessage outHandler ( m, c, commOutMessage ) =
    case commOutMessage of
        Nothing ->
            ( m, c )

        Just (Comm.OutMessage message) ->
            M.evaluate outHandler ( m, c, message )

        Just (Comm.Error e) ->
            let
                ( subModel, subCmd ) =
                    processSystemError m e
            in
            ( subModel, Cmd.batch [ c, subCmd ] )

        Just (Comm.RaiseToast message) ->
            let
                ( toast, nextToastID, cmd ) =
                    Toast.new ToastExpired m.nextToastID message

                updatedToasts =
                    toast :: m.toasts

                updatedModel =
                    { m | toasts = updatedToasts, nextToastID = nextToastID }
            in
            ( updatedModel, Cmd.batch [ cmd, c ] )

        Just (Comm.RouteRequested route newWindow) ->
            let
                opener =
                    if newWindow then
                        Ports.openNewWindowPort

                    else
                        Navigation.pushUrl m.session.key
            in
            ( m, opener (Route.toString route) )


{-| For a given page and current model state, produce a function that takes the output of the page's
update function and lifts it to the corresponding result to return from the top-level update
function.
-}
makePageMapper :
    PageInfo model msg outMsg
    -> Model
    -> ( model, Cmd msg, Maybe (Comm.OutMessage outMsg) )
    -> ( Model, Cmd Msg )
makePageMapper { pageTagger, msgTagger, outHandler } model =
    M.mapComponent (\c -> { model | page = pageTagger c })
        >> M.mapCmd msgTagger
        >> evaluateCommOutMessage outHandler



---- Information records for all subpages.


clusterInfo : PageInfo Page.Cluster.Model Page.Cluster.Msg Page.Cluster.OutMsg
clusterInfo =
    { pageTagger = Cluster
    , msgTagger = ClusterMsg
    , subscriptions = Page.Cluster.subscriptions
    , update = Page.Cluster.update
    , outHandler =
        \msg model ->
            case msg of
                Page.Cluster.SetSlots s ->
                    update (GotSlots s) model
    }


commandListInfo : PageInfo Page.CommandList.Model Page.CommandList.Msg Page.CommandList.OutMsg
commandListInfo =
    { pageTagger = CommandList
    , msgTagger = CommandListMsg
    , subscriptions = Page.CommandList.subscriptions
    , update = Page.CommandList.update
    , outHandler =
        \msg model ->
            case msg of
                Page.CommandList.NoOp ->
                    ( model, Cmd.none )
    }


experimentDetailInfo : PageInfo Page.ExperimentDetail.Model Page.ExperimentDetail.Msg Page.ExperimentDetail.OutMsg
experimentDetailInfo =
    { pageTagger = ExperimentDetail
    , msgTagger = ExperimentDetailMsg
    , subscriptions = Page.ExperimentDetail.subscriptions
    , update = Page.ExperimentDetail.update
    , outHandler =
        \msg model ->
            case msg of
                Page.ExperimentDetail.AuthenticationFailure ->
                    ( model, Cmd.none )
    }


experimentListInfo : PageInfo Page.ExperimentList.Model Page.ExperimentList.Msg Page.ExperimentList.OutMsg
experimentListInfo =
    { pageTagger = ExperimentList
    , msgTagger = ExperimentListMsg
    , subscriptions = Page.ExperimentList.subscriptions
    , update = Page.ExperimentList.update
    , outHandler =
        \msg model ->
            case msg of
                Page.ExperimentList.SetCriticalError errorMessage ->
                    ( { model | criticalError = Just errorMessage }, Cmd.none )
    }


loginInfo : PageInfo Page.Login.Model Page.Login.Msg Page.Login.OutMsg
loginInfo =
    { pageTagger = Login
    , msgTagger = LoginMsg
    , subscriptions = always Sub.none
    , update = Page.Login.update
    , outHandler =
        \msg model ->
            case msg of
                Page.Login.LoginDone user ->
                    ( setSessionUser model (Just user), Cmd.none )
    }


logoutInfo : PageInfo Page.Logout.Model Page.Logout.Msg Page.Logout.OutMsg
logoutInfo =
    { pageTagger = Logout
    , msgTagger = LogoutMsg
    , subscriptions = always Sub.none
    , update = Page.Logout.update
    , outHandler =
        \msg model ->
            case msg of
                Page.Logout.LogoutDone ->
                    ( setSessionUser model Nothing
                    , Navigation.pushUrl model.session.key (Route.toString (Route.Login Nothing))
                    )
    }


notebookListInfo : PageInfo Page.NotebookList.Model Page.NotebookList.Msg Page.NotebookList.OutMsg
notebookListInfo =
    { pageTagger = NotebookList
    , msgTagger = NotebookListMsg
    , subscriptions = Page.NotebookList.subscriptions
    , update = Page.NotebookList.update
    , outHandler =
        \msg model ->
            case msg of
                Page.NotebookList.NoOp ->
                    ( model, Cmd.none )
    }


shellListInfo : PageInfo Page.ShellList.Model Page.ShellList.Msg Page.ShellList.OutMsg
shellListInfo =
    { pageTagger = ShellList
    , msgTagger = ShellListMsg
    , subscriptions = Page.ShellList.subscriptions
    , update = Page.ShellList.update
    , outHandler =
        \msg model ->
            case msg of
                Page.ShellList.NoOp ->
                    ( model, Cmd.none )
    }


tensorboardListInfo : PageInfo Page.TensorBoardList.Model Page.TensorBoardList.Msg Page.TensorBoardList.OutMsg
tensorboardListInfo =
    { pageTagger = TensorBoardList
    , msgTagger = TensorBoardListMsg
    , subscriptions = Page.TensorBoardList.subscriptions
    , update = Page.TensorBoardList.update
    , outHandler =
        \msg model ->
            case msg of
                Page.TensorBoardList.NoOp ->
                    ( model, Cmd.none )
    }


trialDetailInfo : PageInfo Page.TrialDetail.Model Page.TrialDetail.Msg Page.TrialDetail.OutMsg
trialDetailInfo =
    { pageTagger = TrialDetail
    , msgTagger = TrialDetailMsg
    , subscriptions = Page.TrialDetail.subscriptions
    , update = Page.TrialDetail.update
    , outHandler =
        \msg model ->
            case msg of
                Page.TrialDetail.NoOp ->
                    ( model, Cmd.none )
    }


logViewerInfo : PageInfo Page.LogViewer.Model Page.LogViewer.Msg Page.LogViewer.OutMsg
logViewerInfo =
    { pageTagger = LogViewer
    , msgTagger = LogViewerMsg
    , subscriptions = Page.LogViewer.subscriptions
    , update = Page.LogViewer.update
    , outHandler =
        \msg model ->
            case msg of
                Page.LogViewer.NoOp ->
                    ( model, Cmd.none )
    }
