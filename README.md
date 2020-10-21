# goanda-hedger
Hedging Strategies based on goanda library (market order at price() level)

**WARNING: DO NOT RUN ON LIVE ACCOUNTS, NOT TESTED**

**HARD-CODED SUPPORT FOR EUR_USD ONLY**

**THIS SOFTWARE TAKES FULL CONTROL ON ACCOUNT POSITIONS**

## Set-up dev environment


In the root directory of the project (needed to download the `goanda` dependency)

```
export GOPATH=`pwd`
cd src
go get
```

## Set-up and options to run the strategy

Several environment variables are required to run the strategy on an Oanda Portfolio:

Here is an example of set-up for the Oanda practice environment :

```
export OANDA_API_URL="https://api-fxpractice.oanda.com"
export OANDA_STREAM_URL="https://stream-fxpractice.oanda.com"

export OANDA_API_KEY="YOUR_API_PROVIDED_OANDA_FOR_YOUR_ACCOUNT"
export OANDA_ACCOUNT="YOUR_SUB_ACCOUNT_NUMBER"
```

### Running from an empty state (no active Hedge):

```
go run main.go
```

### Add a static Hedge in the Manager:

`POST` a static Hedge for a 10,000 EUR_USD short:

```
curl -data '{"Size": -10000}' localhost:8080/hedge/static
```

### Add a dynamic Hedge in the Manager:

`POST` a dynamic Hedge for 1,000 EUR_USD position leaps at 0.1% price steps:

```
curl -data '{"Size0": 1000, "Scale": 0.1}' localhost:8080/hedge/dymanic
```

### Save the state of the Hedges Inventory for backup:

```
curl localhost:8080/hedge >state.json
```


### Restart the Application from a backup file:

```
go run main.go -statefile state.json
```
