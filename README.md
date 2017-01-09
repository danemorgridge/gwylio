# Gwylio

![Build Status](https://api.travis-ci.org/emoneyadvisor/gwylio.svg?branch=master)

Gwylio is a monitoring tool for Elasticsearch, similar to the Marvel plugin. It works by querying the Elasticsearch cluster for realtime statistic data and then indexing those results back into Elasticsearch to provide historical record of your clusters performance. You can then setup dashboards in Kibana to visualize the data.

It runs as a separate process, and not a plugin, which gives the extra benefit of being able to act independently of the Elasticsearch cluster. Once alerting is in place, it will be able to send you notifications if the cluster becomes unresponsive. It also means that it doesn't have be installed on every node in the cluster. It also allows you to monitor multuple clusters and send the stat data to a single central cluster.



## Installation

To use Gwylio, you can either download a build from the [Releases](https://github.com/emoneyadvisor/gwylio/releases) page, or download the source (see: [Working with the source](#workingwithsource))



Unzip the package to where you will run it and modify the configuration file:

```yaml
# comma separated list of hosts to collect data from
elastic_clients_from:
  - hosts: ["http://localhost:9200"]
    cluster_name: "my-cluster"
    expected_node_count: 2
# comma separated list of hosts to send data to
elastic_clients_to: ["http://localhost:9200"]

#indexes will be patterned {prefix}-2006.01.02
index_prefix: ".gwylio"

# interval in seconds to query elasticsearch for node and cluster stats
collect_interval: 30

notify_on_node_count_change: true
notify_on_cluster_yellow: true
notify_on_cluster_red: true
notify_on_cluster_unavailable: true

notifications: ["slack"]

# slack webhook uri for notifications (can be overridden by each task)
slack_webhook_uri: ""
slack_webhook_channel: ""
slack_webhook_sender: ""
slack_webhook_emoji: ""

smtp_server: ""
smtp_port: 25
smtp_auth_user: ""
smtp_auth_password: ""
smtp_from_address: ""
smtp_to_addresses: [""]

hipchat_auth_token: ""
hipchat_base_url: ""
hipchat_room: ""
```

The `elastic_clients_from` setting is an aray of an array of cluster settings including an array of URIs that point to the HTTP accessable client nodes for your Elasticsearch clusters, the cluster name (as defined in the elasticsearch configuration files), and the number of nodes in the cluster.

If you had two clusters with three nodes each, your setting might look like this:

```yaml
elastic_clients_from:
  - hosts: ["http://10.0.0.12:9200","http://10.0.0.13:9200","http://10.0.0.14:9200"]
    cluster_name: "cluster-one"
    expected_node_count: 3
  - hosts: ["http://10.0.0.42:9200","http://10.0.0.43:9200","http://10.0.0.44:9200"]
    cluster_name: "cluster-two"
    expected_node_count: 3
```

Each cluster is monitored separately and the URIs will be queried in the order they are listed in the configuration file. The subsequent URIs will only be qureried in the event that the first URI is unavailable or returns a result other than `200` or if you have dedicated client nodes. With nodes that only have the client role, Elasticsearch will not give you node statistic data on the other client only nodes in the cluster, so they must be queried independently. It is important for this reason to make sure that all of your client only nodes are listed in the configuration if you want statistic on them. 

The `elastic_clients_to` setting is the cluster that you will be indexing data into. This is an array of URIs and has the same failover concept as the `elastic_clients_from ` seetting, but is only for one cluster, not multuple.

The `index_prefix` setting is for determining what your resulting indicies will be created as. Gwylio creates daily indicies with the the configured prefix and date. For example, the data indexed on August, 4th, 2016 would go into the .gwylio-2016.08.04 index. You can set the prefix to be anything you want as long as it is a valid Elasticsearch index name (leading underscores are invalid, for instance). A leading period (`.`) tells Elasticsearch that this is a special index and doesn't show up by default in plugins like Kopf with out selecting the option to show special indexes. The default of using a leading period is to separate the index from your normal data, but it is not required.

The `collect_interval` setting tells Gwylio how often, in seconds, you would like to query the Elasticsearch cluster for data.

There are four setting that control internal cluster notifications. This data is already avaiable from the cluster and node statistics calls, so separate queries are not sent.

`notify_on_node_count_change` will immediately notify you if the the node count on a cluster changes, letting you know quickly if a node leaves the cluster.

`notify_on_cluster_yellow` and `notify_on_cluster_red` will notify you if the cluster state changes to either yellow or red respectively.

`notify_on_cluster_unavailable` will send a notification if the cluster cannot be reached on any of the configured URIs.

The `notifications` array is a string of notification types that you wish to use. Currently only Slack notifications are supported, but email, hipchat, and others will be incorporated in the future.

### Slack Notifications

There are four settings that control how Slack notifications get sent.

`slack_webhook_uri` is the only required setting. When you configure a Slack webhook, you will be provided with a URL that is used to send the message to Slack. This URL is configured with the channel, sender name, and emoji icon. These are overridable with the following settings:

`slack_webhook_channel` is the channel that you wish to post the message to. It can also be a user instead of an actual channel. If a user is selected (@someuser), the message will show up in their slackbot message area.

`slack_webhook_sender` is the name of the sender. This is the name that will appear along with the message.

`slack_webhook_emoji` is the emoji icon that will used for the notification. Any emoji that is configured within your slack team. 

### Email Notifications

`smtp_server` is the host name or IP address of the SMTP server to send mail through.

`smtp_port` is the port the SMTP server is listening on.

If you are using SMTP Authentication, set the `smtp_auth_user` and `smtp_auth_password` to the proper credentials.

`smtp_from_address` is the email address the email will be sent from.

`smtp_to_addresses` is an array of email addresses that will receive the email notification.

### Hipchat Notifications

`hipchat_auth_token` is the auth token from HipChat.

`hipchat_base_url` is the base url for the HipChat server. If you are not using a custom HipChat server, leave this blank.

`hipchat_room` is the room you want the notification to go to.

## Alerting

Gwylio has built in alerting to notify you of cluster state and node counts, but you can write custom alerting rules.

Each rule is defined as a json document stored in the `rules` folder along side the Gwylio binary.

### Rule types

There are two types of rules, `count` and `search`.

A count rule will send a count query and the number of documents that match the query will be returned and not the actual documents.

A search rule will send a search query and the number of documents will be returned, as well as the documents themselves. When email notifications are implemented, the result set will be sent as an attachment to the email. Examples for both are in the rules folder in source.

An example count rule:

```json
{
    "rule_name": "Example Count Rule", 
    "rule_type": "count",
    "notification_message": "Example Rule was hit", 
    "cluster_name":"my-cluster",  
    "index_name":".gwylio-*",
    "document_type":"",
    "enabled": false,
    "operator": ">", 
    "threshold": 10,
    "interval":5,
    "notification_interval": 60,
    "notification_overrides": {        
        "slack": {
            "uri":"",
            "channel":"",
            "sender":"",
            "emoji":""
        },
        "email": {
            "smtp_server":"",
            "smtp_port":25,
            "smtp_auth_user":"",
            "smtp_auth_password":"",
            "smtp_from_address":"",
            "smtp_to_addresses":[""]
        }
    },    
    "query": {
        "query": {
            "filtered": {
                "filter": {
                    "range": {
                        "timestamp": {
                            "from": "now-6h",
                            "to": "now"
                        }
                    }
                }
            }
        }
    }
}
```

The `rule_name` setting is used for identification and will only be seen in the logs.

The `rule_type` must be either "count" or "search".

The `notification_message` will be the actual message that is posted or emailed. The counts found will be appended to this message.

To identify the cluster, the `cluster_name` property must match the configured cluster name, and it must be one of the clusters configured in the gwylio.yml file.

The `index_name` and `document_type` refer to the actual Elasticsearch index name and document type that you wish the query to run against. They will be used to build the URL for the query. They are both optional however. If you provide a blank index, the query will run on all indexes. You can use wildcards or aliases, just like with any Elasticsearch query. If the document type is provided, it will be used as part of the query. If it is left off, all document types will be queried.

Rules can be enabled or disabled by settin the enabled flag to either `true` or `false`.

The `operator` setting goes with the `threshold` setting to determine the count of items that will trigger an event. In this example, a count greater than 10 will result in the notification being sent. The allowed operators are as follows:

* Greather than: "gt" or ">"  
* Greather than or equal to: "gte" or ">=" 
* Less than: "lt" or "<" 
* Less than or equal to: "lte" or "<=" 
* Equal to: "eq" or "==" 
* Not equal to: "neq" or "!=" or "<>"

The `interval` setting governs how often (in minutes) the query will run.

The `notification_interval` is used to set how often a notification should be sent (in minutes) if the condition is still met. For example, if your query matches on every run for two hours, you'll get a notifications every hour with the option set to `60`.

If you want to override any of the notification setting options from the main configuration you can set those here. Different people might be interested in different data. For Slack notifications, any of the four options can be overridden and all are optional. If no override is configured, the base settings will be used.

Lastly, the `query` option is the actual query that will be sent to Elasticserarch. The query is sent as-is and no manipulation will be done to it. The example query looks at all documents over the last 6 hours. [Elasticsearch date math](https://www.elastic.co/guide/en/elasticsearch/reference/current/common-options.html#date-math) makes building time-based queries that don't require any hard coding of times or additional manipulation of the query.


## <a name="workingwithsource"></a> Working with the source

The first thing you need to do is make sure you have [Go installed](https://golang.org/doc/install) and have your `GOPATH` set properly. You're `GOPATH` is a path you set on your machine, not one that is already there. It is also used for all projects, so it needs to be centrally located.

Once you have go installed, clone the source code into the following directory:

`$GOPATH/src/github.com/emoneyadvisor/gwylio`

`cd` to that directory, then run `go get ./...` to download all dependencies.

Executing `go build` from that directory will build a local gwylio file in the same folder. (gwylio.exe on Windows)

Executing `go install` from that directory will buil gwylio.exe in the `$GOPATH/bin/` folder.


