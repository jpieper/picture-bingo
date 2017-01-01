import Html exposing (Html, button, div, text, ul, li, img, input, form)
import Html.Events exposing (onClick, on)
import Html.Attributes exposing (class, src, action, method, type_)
import Http
import Json.Decode as Decode
import FileReader


main =
  Html.program { init = init, view = view, update = update, subscriptions = subscriptions }


-- MODEL

type alias Model =
    {
        cardName : String,
        value : Int,
        details : CardDetails
    }

type alias CardDetails =
    {
        pictures : List Picture
    }

type alias Picture =
    {
        name : String,
        url : String
    }

type alias Files =
    List FileReader.NativeFile


init = (Model "" 5 (CardDetails []), getCardName)

-- UPDATE

type Msg = Increment
         | Decrement
         | NewCard (Result Http.Error String)
         | NewCardDetails (Result Http.Error CardDetails)
         | FileUpload Files
         | UploadComplete (Result Http.Error Decode.Value)

update : Msg -> Model -> (Model, Cmd Msg)
update msg model =
  case msg of
    Increment ->
      ({ model | value = model.value + 1 }, Cmd.none)

    Decrement ->
      ({ model | value = model.value - 1 }, Cmd.none)

    NewCard (Ok cardName) ->
      let
          newModel = { model | cardName = cardName}
      in
          (newModel, getCardDetails newModel)

    -- Ignore errors for now
    NewCard (Err _) ->
      ({ model | cardName = (toString (Err))}, Cmd.none)

    NewCardDetails (Ok details) ->
        ({ model | details = details }, Cmd.none)

    NewCardDetails (Err _) ->
        (model, Cmd.none)

    FileUpload (files) ->
        case List.head files of
            Just file ->
                (model, sendFileToServer model file)
            Nothing ->
                (model, Cmd.none)


    UploadComplete (Ok _) ->
        (model, Cmd.none)

    UploadComplete (Err _) ->
        (model, Cmd.none)


-- VIEW

view : Model -> Html Msg
view model =
  div []
    [ div [] [ Html.h2 [] [ text model.cardName ] ]
    , ul [ class "picture-list" ] <| List.map viewPicture model.details.pictures
    , input [ type_ "file", onchange FileUpload ] []
    , button [ onClick Decrement ] [ text "-" ]
    , div [] [ text (toString model) ]
    , button [ onClick Increment ] [ text "+" ]
    ]

viewPicture picture =
    li [] [ text (toString picture.name)
          , img [ src picture.url ] [] ]

onchange action =
    on
        "change"
        (Decode.map action FileReader.parseSelectedFiles)

-- SUBSCRIPTIONS

subscriptions model =
    Sub.none

-- HTTP

getCardName =
    let
        url = "/v1/make_new_card"
    in
        Http.send NewCard (Http.get url decodeCardName)

decodeCardName =
    Decode.at ["name"] Decode.string

getCardDetails model =
    let
        url = "/v1/get_card/" ++ model.cardName
    in
        Http.send NewCardDetails (Http.get url decodeCardDetails)

decodeCardDetails : Decode.Decoder CardDetails
decodeCardDetails =
    Decode.at ["pictures"] (Decode.map CardDetails (Decode.list pictureDecoder))

pictureDecoder : Decode.Decoder Picture
pictureDecoder =
    Decode.map2 Picture
        (Decode.field "name" Decode.string)
        (Decode.field "url" Decode.string)

sendFileToServer : Model -> FileReader.NativeFile -> Cmd Msg
sendFileToServer model buf =
    let
        body =
            Http.multipartBody
                [ FileReader.filePart "file" buf ]
    in
        Http.post ("/v1/add_picture/" ++ model.cardName) body Decode.value
            |> Http.send UploadComplete
