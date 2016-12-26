import Html exposing (Html, button, div, text)
import Html.Events exposing (onClick)
import Http
import Json.Decode as Decode


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
        pictures : List String
    }


init = (Model "" 5 (CardDetails []), getCardName)

-- UPDATE

type Msg = Increment
         | Decrement
         | NewCard (Result Http.Error String)
         | NewCardDetails (Result Http.Error CardDetails)

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


-- VIEW

view : Model -> Html Msg
view model =
  div []
    [ div [] [ Html.h2 [] [ text model.cardName ] ]
    , button [ onClick Decrement ] [ text "-" ]
    , div [] [ text (toString model) ]
    , button [ onClick Increment ] [ text "+" ]
    ]

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
        url = "/v1/get_card?card=" ++ model.cardName
    in
        Http.send NewCardDetails (Http.get url decodeCardDetails)

decodeCardDetails : Decode.Decoder CardDetails
decodeCardDetails =
    Decode.at ["pictures"] (Decode.map CardDetails (Decode.list Decode.string))
