# ResourceConsist
**ResourceConsist** ([official site](https://kusionstack.io/docs/operating/manuals/resourceconsist)) aims to make a 
customized controller can be realized easily, and offering the ability of following 
[**PodOpsLifecycle**](https://kusionstack.io/docs/operating/concepts/podopslifecycle) for controllers.

The only thing users need to do to realize a customized controller is writing an adapter implementing 
[ReconcileAdapter](https://github.com/KusionStack/resourceconsist/blob/main/pkg/frame/controller/types.go#L67). 
For controllers following **PodOpsLifecycle**, 
[WebhookAdapter](https://github.com/KusionStack/resourceconsist/blob/main/pkg/frame/webhook/types.go#L26) is 
also necessary to be implemented.

<img src="docs/resourceconsist.png" width="500" height="400"/>

🤠**Employer** is the entity responsible for managing and coordinating the utilization of another resource, 
similar to how a service selects and controls pods.

👩‍💻**Employee** is the resource managed by another resource, like pods selected by service.

👉 Please refer to [key concepts](https://github.com/KusionStack/resourceconsist/tree/main/docs/keyconcepts.md) 
to find out what 🤠**Employer**/👩‍💻**Employee**/... are.

## 💻 Get Started
### 🔧 Tutorial
**ResourceConsist** offers the frame for starting controller. 
Controllers started by adapters and resource consist frame can handle resources beyond or in cluster by themselves 
or cooperate with other controllers.

[DemoControllerAdapter](https://github.com/KusionStack/resourceconsist/tree/main/pkg/frame/controller/resourceconsist_controller_suite_test.go#L31)
is an adapter handles resources by itself, while [SlbControllerAdapter](https://github.com/KusionStack/resourceconsist/tree/main/pkg/adapters/alibabacloudslb/alibabacloudslb_controller.go#L35)
is an adapter handles resources cooperate with CCM controller.

With **ResourceConsist**, you can build a bridge between existing controllers and PodOpsLifecycle, just like what
SlbControllerAdapter](https://github.com/KusionStack/resourceconsist/tree/main/pkg/adapters/alibabacloudslb/alibabacloudslb_controller.go#L35) did.

Please visit [tutorial](https://github.com/KusionStack/resourceconsist/tree/main/docs/tutorial.md) to start a controller.
## ☎️ Contact us
- Twitter: [KusionStack](https://twitter.com/KusionStack)
- Slack: [Kusionstack](https://join.slack.com/t/kusionstack/shared_invite/zt-19lqcc3a9-_kTNwagaT5qwBE~my5Lnxg)
- DingTalk (Chinese): 42753001
- Wechat Group (Chinese)

  <img src="docs/wx_spark.jpg" width="200" height="200"/>
## 🎖︎ Contribution guide
**ResourceConsist** is currently in its early stages. Our goal is to make a customized controller can be realized 
easily, especially for controllers following PodOpsLifecycle. 

We will continue implementing more common used traffic controller into 
[adapters](https://github.com/KusionStack/resourceconsist/tree/main/pkg/adapters)

🚀 If you want to contribute to built-in adapters, you can start from contribute them into 
[experimental/adapters](https://github.com/KusionStack/resourceconsist/tree/main/pkg/experimental/adapters). 
We will move it into [adapters](https://github.com/KusionStack/resourceconsist/tree/main/pkg/adapters) when the 
experimental adapters are ready to release.

We welcome everyone to participate in construction with us. Visit the [contribution guide](docs/contribution.md)
to understand how to participate in the contribution KusionStack project.
If you have any questions, please [submit the issue](https://github.com/KusionStack/resourceconsist/issues).
