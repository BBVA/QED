## Introduction

QED implements a forward-secure append-only persistent authenticated data structure.
Each append operation produces as a result a cryptographic structure (a signed snapshot), which can verify:

 * whether or not a piece of data is on QED
 * whether or not the appended data is consistent, in insertion order, to another entry

To verify both statements, we need the snapshot, the piece of data inserted and a QED proof.
QED emits a snapshot on every event insertion, and they need to be accessible elsewhere.
Also QED does not verify nor emit unsolicited proofs, it’s the user responsibility to know when and how to verify the data depending on their needs.

Last, QED does not store the data itself, only a representation of it produced by a collision-resistant hash function.
QED does not provide means to map a piece of data to a QED event, so the semantic of the appended data and the relation between each item appended is also a client responsibility.

We define QED event as the result of applying a SHA256 hash function to a piece of data generated for its inclusion in QED. Each event has a temporal relation between the next and the prior one. QED represents the temporal relation by an always increasing 64-bit integer called QED version.

All the scripts and code described live in the branch ‘tampering’ in the main repository.

## Why?

There are multiple technologies to achieve a similar functionality as QED, such as signed data in a database, or block chain’s related structures.

The advantages of the data structure QED implements are:

 * scalability to reach thousands of millions of events
 * proof of membership or non-membership generation in logarithmic time
 * proofs of logarithmic size
 * proof of temporal consistency related to QED insertion time

## Architecture and components

In every QED scenario there might be one or more components of this nature:

 * Events source: something that stores the data needed to build QED events
 * Application: a system/action that works with data from the event source to its job
 * Third party services: a system/action used by application or event source to execute their job
 * QED log: where the authentication data lives, normally a RAFT replicated cluster
 * QED gossip agents: network on which QED emits snapshots to agents
 * QED snapshot store: stores all the snapshots generated by QED

The relation between these components will affect the deployment architecture as there are multiple valid alternatives which guarantee a correct QED operation and service.

The relation between the QED log, gossip agents and snapshot store depends on their connectivity requirements and APIs:

 * QED log receives events using an HTTP, and a protobuf based GRPC API
 * QED log emits via a gossip UDP network messages processed by the QED agents
 * QED publisher agent pushes snapshots to the store using a given HTTP API.

The relation between the event source, the third party services and the application does not affect the QED per se, but the application must be able to map an event to a QED event univocally, because QED Log stores no data but a representation in the form of a SHA256 hash.

The application needs to talk to QED Log, either directly by its API or using one of the QED Log supported client libraries.

Please refer to the QED manual to knows about how to deploy a production instance.

## Tamperings

### Event source

 * QED can emit a verifiable proof to check if an event is on QED
 * QED can emit a verifiable proof to check if two events are consistent with each other in the order of insertion
 * The user has the responsibility to ask for these proofs and verify them and can user the QED gossip network to build auditors and monitors adapted to its use case.
 * The user should use a secret unknown to QED for the QED event mapping function
 * QED does not audit nor emits proof or verify proactively any event
 * QED does not alert in real time about event source changes

### Application

 * We cannot guarantee an application will use QED
 * We can use QED capabilities to build external tooling to check the application expected behaviour

### Third party

 * We can use QED to verify changes in third-party data source using a QED client which must implement a mapping function between the third-party data to QED events.
 * We can use QED to check the history of changes of a third party ordered data source. Also, the source of the order could be build from another means.

### QED log

QED is resistant to naïve attempts to tamper with its database. A modification of a single leaf of a tree, or path is detected easily. This kind of tampering tries to harm the credibility of the system by making it complain or to avoid the validation of a correct event. *Once the QED is tampered with, there is no rollback. Only a complete rebuild can ensure its coherence.*

We can alter the events stored in QED in a way that the proofs will verify only if the QED version is reset to an old version and we insert events from that version again using the QED append algorithm to regenerate all the intermediate nodes of the trees:


    v0————>v1————>v2————>v3 ————>v4 ————> v5              original history
                         |                                version reset
                         |—>v3’————>v4’————>v5’————>v6    forked history

This a theoretical attack,  in practice it is unfeasible to do such an attack without being detected,  as it requires modifying a running program which replicates on real time, without being noticed.
Also, even if the attack happens, it can be detected doing a full audit checking all events against the event source and the snapshot store.

*QED will not know which component was tampered, only that an event being check has either its event source, its snapshot or its QED event altered. We will not establish the source of truth unless we do a full audit which comprises the insertion of all QED events again to regenerate the source, the log and the snapshots to check the differences.*

To further protect a QED deployment against such tampering, we recommend salting the QED events with a secret (which QED does not know) verifiable by the event stakeholders and recommends implementing a monitoring agent that check the snapshot store searching for duplicate QED versions.

Another recommendation is to make QED clusters to block any arbitrary nonauthenticated joins, replications or from-disk recoveries.

Last, the teams or companies in charge of the QED log, agents and snapshot store should be different to avoid collusion.

### QED agents

The gossip agents can use certificates and/or user/password to authenticate against each other.

The agent's mission is to check the QED activities to identify anomalous behaviours and also publish the snapshots into the store.

They can be tampered as any other application, making them publish altered snapshots or to omit any functionality.
But the QED proofs verification process will detect modifications regarding the events being checked as long as the event source or QED log are untampered.

### QED snapshot store

The snapshot store can be compromised to change snapshots stored, but like in the QED agents case, the QED proofs verification process will fail as long as the event source or QED log are untampered.

## Use cases

We describe practical uses of QED in different technological or business areas to illustrate what can and cannot do.

### Trusting third-party services: the case of dependency management

A team of developers works on a software project stored in a repository. An automated software construction pipeline builds and packages the work of the team. It downloads multiple third-party repositories containing software or artifacts needed in the build pipeline.

A dependency verification service scope could go from a single team to the entire world. As long as the metadata used in the QED events satisfies its users.

How can we check if a dependency has been tampered?

 * the event source will be the source code repository, it will trigger the pipeline on each commit
 * the third-party service will be the dependency distribution service
 * the application will be the pipeline construction software

We can leverage QED to verify the history of an artifact ensuring our dependencies are correct, failing the construction in the pipeline in case one dependency has changes.

In this scenario we can also contemplate multiple teams working on multiple software projects with overlapping dependencies which uses a single QED log and snapshot store.

For this example, we will suppose for each dependency we have the following data in JSON format:

    {
        “pkg_name”: “name”,
        “pkg_version”: “vX.XX.X”,
        “pkg_digest”: “xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx”
    }


The general workflow can comprise the following steps:
 a) develop code, adding dependencies
 b) ensure those dependencies are correct
 c) add the verified dependencies to the QED log
 d) trigger the pipeline to build the software
 e) download the dependencies
 f) generate a QED event for each dependency
 g) verify each generated entry against QED, failing when the event does not verify

#### Tampering the dependency repository

Someone compromises the dependency distribution system to inject malicious code into its users projects.

Our pipeline code will try to download the dependency from the official location, and will execute the steps from d) to g).
The verification process will fail if there is a change in the QED event generated from the third party repository data.
The mapping function between the third-party data and QED must work in a way that maps univocally all the relevant information into a QED event, making any change clear (a hash function for example).

##### The happy path

    $ ./generate.sh
    usage:
    ./generate.sh pkg_name pkg_version pkg_url
    $ ./generate.sh qed 0.2 https://github.com/BBVA/qed/releases/download/v0.2-M3/qed_0.2-M3_linux_amd64.tar.gz > newentry.json
    Package
    {
        “pkg_name”: “qed”,
        “pkg_version”: “0.2”,
        “pkg_digest”: “5f6486ff5d366f14f9fc721fa13f4c3a4487c8baeb7a1e69f85dbdb8e2fb8ab2”
    }
    saved in /tmp/tmp.GS5kwBUYJu
    $ ./append.sh newentry.json
   
    Adding key [ {
        “pkg_name”: “qed”,
        “pkg_version”: “0.2”,
        “pkg_digest”: “5f6486ff5d366f14f9fc721fa13f4c3a4487c8baeb7a1e69f85dbdb8e2fb8ab2”
    } ]

    Received snapshot with values:

     EventDigest: f56c757b5403c71ced0773684a259c7a2dcde6e4232b251ceae5084d58ff356e
     HyperDigest: bcca38e67883f492b8dece031290a4b1b5cfa801d9917670f419b183487163be
     HistoryDigest: 784674a832f41ff7b9ddc13bdb2aef2093975319c5f23b69f04fbae163668975
     Version: 0
    
     $ ./membership.sh 0 newentry.json
    
    Querying key [ {
        “pkg_name”: “qed”,
        “pkg_version”: “0.2”,
        “pkg_digest”: “5f6486ff5d366f14f9fc721fa13f4c3a4487c8baeb7a1e69f85dbdb8e2fb8ab2”
    } ] with version [ 0 ]
   
    Received membership proof:
   
     Exists: true
     Hyper audit path: <TRUNCATED>
     History audit path: <TRUNCATED>
     CurrentVersion: 0
     QueryVersion: 0
     ActualVersion: 0
     KeyDigest: f56c757b5403c71ced0773684a259c7a2dcde6e4232b251ceae5084d58ff356e
   
    Please, provide the hyperDigest for current version [ 0 ]: bcca38e67883f492b8dece031290a4b1b5cfa801d9917670f419b183487163be
    Please, provide the historyDigest for version [ 0 ] : 784674a832f41ff7b9ddc13bdb2aef2093975319c5f23b69f04fbae163668975
   
    Verifying with Snapshot:
   
     EventDigest:f56c757b5403c71ced0773684a259c7a2dcde6e4232b251ceae5084d58ff356e
     HyperDigest: bcca38e67883f492b8dece031290a4b1b5cfa801d9917670f419b183487163be
     HistoryDigest: 784674a832f41ff7b9ddc13bdb2aef2093975319c5f23b69f04fbae163668975
     Version: 0
   
    Verify: OK

##### Lifecycle of a single dependency: tampering the source of a past release

Create a timeline of a single dependency, for example, for Facebook database Rocksdb, we add the following versions to QED:
   
    v5.13.3
    v5.12.5
    v5.13.4
    v5.14.2
    v5.14.3
    v5.15.10
    v5.16.6
    v5.17.2
    v5.18.3

in order from old to new.

Generate a data entry for each version and add it to QED:

    $ for i in v5.13.3 v5.12.5 v5.13.4 v5.14.2 v5.14.3 v5.15.10 v5.16.6 v5.17.2 v5.18.3; do
        ./generate.sh rocksdb ${i} https://github.com/facebook/rocksdb/archive/${i}.zip > rocksdb/${i}.json
    done

Append the data to a QED server:
   
    $ for i in v5.13.3 v5.12.5 v5.13.4 v5.14.2 v5.14.3 v5.15.10 v5.16.6 v5.17.2 v5.18.3; do ./append.sh rocksdb/$i.json; done

Now we have a QED with 9 rocksdb versions ordered by release date from older to newer.

Also we have 9 snapshots in a third party public store, so we can get anytime the snapshot of each version.

In this example we suppose we are using the version v5.16.6 to build our project. Someone compromises the dependency distributor, a new package has been uploaded to the official repository, with an old version but new contents and new download hashes.

Ir our pipeline, every time we download a dependency, we generate its QED entry, but this time, because someone altered the package, we generate a different version of the entry:

    rocksdb/corrupted-v5.16.6.json
   
Before using it to build our software, we ask QED for a membership proof, so we can verify that our download verifies the QED membership proof:

    $ ./membership.sh 6 rocksdb/corrupted-v5.16.6.json

To verify the event, we need the correct snapshot information. Because we’re using the interactive client, it ask us the hyperDigest for the current version of the QED. We go to the snapshot store to get it:

    $ ./getsnapshot.sh 8
   
We use the hyperdigest presented here, and the client tries to verify the information, but it fails.

As we can see, the QED tells us that the information was not on QED and the client verified that there is no such event given the cryptographic information published in the insertion time of the event. With this information we can alert the alteration of one of our dependencies and stop the pipeline alerting the devops team of the issue.

We have authenticated a third party repository (a github release repository) which is the source of the events, then we have inserted into QED the information related to a set of verified releases of rocksdb. Later, the repository was the target of an attack and a changed version of a dependency was downloaded by the software construction pipeline. Our dependency check phase checked the information against QED and discovered a tamper in the remote software repository.

Storing the hashes into the repository and checking against them when downloading the dependency does the same job, cheaper. Also, most package management tools like go mod, npm, cargo, etc. use dependency package files containing hashes of the dependency version that must be used in the construction.

But we must note having an external validation tool is useful in situations when we need resistance to tampering and separation of duties.

Last, as a reference example, in the case of [the event-stream npm case](https://snyk.io/blog/malicious-code-found-in-npm-package-event-stream/), their source code repository was compromised, and a new version was generated with malicious code in it. In this case, QED will not help as a new version was published so a reviewer must validate the dependency before its insertion in QED.


##### Tampering the source: github code repository

Using the last scenario as a starting point, we have now the situation on which our source code repository has been compromised and a new download URL and hash has been provided to our package manager.

Here, our pipeline will download the new dependency and will generate the entry for QED. But this entry was not in QED, so the check will fail the verification process.

This scenario assumes that only authenticated developers can insert entries to QED, and with their digital signature they provide a personal warranty the entry is legit. If someone inserts a malicious event in QED it will generate valid cryptographic information about it, and will verify it was inserted in that time.

Also, QED do not detect vulnerabilities, nor validate the correctness of changes in software,  but can support a process involving multiple parties to be more resistant to them.

Last, we can use QED to store the behaviour of external services, adding the history of changes a service present to its user into QED log and making possible to verify that behaviour regarding its insertion time.

#####  Tampering the application: builder pipeline

Here, an insider compromises the builder system, and instead of building the software as programmed, it will build a special release containing arbitrary code and dependencies.

QED does not verify the proofs it emits, and if a client of QED does not take into consideration the output of its own QED proof verification process, there is nothing QED can do.

We can leverage the gossip agents platform included in QED to build external tools. For example a special proxy to detect unwanted dependencies downloads. This proxy server will be in charge of outgoing HTTP connections to the internet and will check against QED all the package URLs before being downloaded.

#####  Tampering the QED log

The QED server log stores its cryptographic information in a local database. This database is replicated against the other nodes of a QED cluster using the RAFT protocol.

For the realization of these tests, we have developed a special QED server which includes a special endpoint to change the database without using the QED API or stopping the server which would be the worst scenario in case of an attack against QED.

In QED we can change either the hyper or the history tree. The hyper tree is a sparse Merkle tree with some optimizations that maps a piece of data with is the position in the history tree. This history tree is an always growing complete binary tree and stores the entries ordered by its insertion time.

The hyper tree will tell us if an event changed or not and will give us a proof which we can use to verify it. The history tree will tell us if the event is consistent with the stored order of another entry, and will give us a verifiable proof.

Starting with the Rocksdb example, we can create a fork in the QED history if we insert new events into QED, into the same positions of old events, and generate new snapshots for them.

It will not work changing the hyper tree only as the history will contain the correct hash and the membership query will not validate. Also, we can’t change just the history leaf in its storage because that will not update all the intermediate nodes, and the generated audit paths will be unverifiable.

A way to rewrite the QED is to insert new events after resetting the balloon version to the one we want to change. In a RAFT cluster, the hijacked node needs to preserve the raft log order to tamper the data without corrupting the RAFT server making it crash. We also need to change all the events from that point in history to the end to have the same history size than before. To reset the RAFT replicated write-ahead-log we need to stop the whole cluster, replicate the desired state, and start node by node again.

We insert into QED three new events, v5.16.6’ v5.17.2’ v5.18.3’, but instead of using the regular API, we will suppose we have a custom QED implementation that inserts in the hyper tree the version we want.

    $ ./resetVersion.sh 6
    $ ./append.sh rocksdb/v5.16.6p.json
    $ ./append.sh rocksdb/v5.17.2p.json
    $ ./append.sh rocksdb/v5.18.3p.json
   

Our forked history tree now looks like:

    […]————> v5.15.10————> v5.16.6’————> v5.17.2’————> v5.18.3’
  

In this situation we will download the compromised dependency, v5.16.6’ and ask QED for it, like we did before:

    $ ./membership.sh 6 rocksdb/v5.16.6p.json

And it will verify it.

To detect this,  we can check if we have two entries of the snapshot in the snapshot store. Also, we can implement processes attached to the gossip network to test if a snapshot has been seen twice with the same version but different hashes or with a time delay bigger than a threshold.

#####  Tampering the QED snapshot store

QED does not provide an implementation for the snapshot store, just the HTTP API to store and retrieve snapshots.

Because the tampering procedure explained before, we advise to select an append only database with an auto-generated primary key, so it is possible to detect multiple snapshots for the same version.

In case someone deletes an entry from the snapshot store, the event inserted in that version won’t validate. And the QED will not issue the same snapshot twice.

Also if someone can control the log and the snapshot store, it is possible to fake a verifiable history, but will fail the verification of the event source. The only way to avoid detection is to control all the parts of the system: the event source, the QED log and the snapshot store. This is equivalent to deploy a new QED with a custom event history inserted.

### Trusting peers through transparency: moving money between different institutions

Multiple institutions transfer money between them, and to avoid discrepancies in their accounting processes they build want a system to verify the transactions made.

 * multiple entities, each one with its own QED log
 * a common shared QED snapshot store
 * an application which will generate a QED entry for each money transfer order
 * an accountability process

How can an entity be sure a transfer order goes before or after another transaction? How can an entity verify the validity of a transaction order?

 * the event source will be the transfer orders of each institution and its associated metadata
 * the application will be the related bank applications that operate the transfer orders
 * the third-party will be the snapshot store shared between all the participants

An example of a procedure could be:
 a) institution x add a transfer order to its QED log
 b) institution x send a transfer order to institution y
 c) institution y receives a transfer order from institution x and queries x QED to verify the order
 d) institution y validates the order and include the verification in its own QED log
 e) institution x validates its transfer is done when the verification of institution y its in the y QED log


##### Tampering the source
##### Tampering the application
##### Tampering the QED log
##### Tampering the QED snapshot store

### Trusting humans: user activity non-repudiation

A client of Bank A acquires new banking products. Those products are subject to risks. And at some point the client decides to accuse the Bank A of cheating in the operations he did. How can the Bank A prove he is lying?

* the event source will be the client signed contracts and the accounting changes and it reasons
* the application will be the related bank applications that operates and generates the accounting data and its changes

##### Tampering the source
##### Tampering the application
##### Tampering the QED log
##### Tampering the QED snapshot store


