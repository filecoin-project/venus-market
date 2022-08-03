## account

`venus` 的 `account` 体系中，各角色的关系如下：

- 一个account对应一个token；
- 一个account管理多个miner；

### 消息推送

```
PushMessage(ctx context.Context, msg *shared.Message, meta *types.SendSpec) (string, error)
```

`venus-messager` 读取 `venus-auth` 验证 RPC 请求时写入的 `accountKey` （`token` 对应的 `account`）,签名时通过 `venus-gateway` 找到 `account`  对应的 `venus-wallet` 服务签名。

```
WalletSign(ctx context.Context, account string, addr address.Address, toSign []byte, meta venusTypes.MsgMeta) (*crypto.Signature, error)
```

### 出块签名
 
`venus-miner` 从 `venus-auth` 获取 `account-miners` 列表, 签名时获取 miner 对应的 account， 调用 `WalletSign` 接口。

## market 中消息处理

```
func (msgClient *MixMsgClient) PushMessage(ctx context.Context, p1 *types.Message, p2 *types.MessageSendSpec) (cid.Cid, error) {
	// venus/lotus fullnode 处理消息
	msgid, err := utils.NewMId()
	if err != nil {
		return cid.Undef, err
	}
	if msgClient.addrMgr != nil {
		fromAddr, err := msgClient.full.StateAccountKey(ctx, p1.From, types.EmptyTSK)
		if err != nil {
			return cid.Undef, err
		}
		account, err := msgClient.addrMgr.GetAccount(ctx, fromAddr)
		if err != nil {
			return cid.Undef, err
		}
		_, err = msgClient.messager.ForcePushMessageWithId(ctx, account, msgid.String(), p1, nil)
		if err != nil {
			return cid.Undef, err
		}
	} else {
		//for client account has in token
		_, err = msgClient.messager.PushMessageWithId(ctx, msgid.String(), p1, nil)
		if err != nil {
			return cid.Undef, err
		}
	}

	log.Warnf("push message %s to venus-messager", msgid.String())
	return msgid, nil
}
```

可以看出：如果 `venus-market` 存在 `miner manager` 时，是通过消息的 `From` 来找对应的 `account` 的， 这个时候就存在一个问题，`miner manager` 初始设计是从 auth 获取的，是
`miner-account`, 而不是 `from-account`。

为了解决这个问题，目前的实现是：

（1） 从 auth 获取的 `miner-account` 进行扩展，举例说明：
```
miner：f0128788, 
owner: f01630430(多签地址)，
worker：f3wylwd6pclppme4qmbgwled5xpsbgwgqbn2alxa7yahg2gnbfkipsdv6m764xm5coizujmwdmkxeugplmorha,
control: f3r47fkdzfmtex5ic3jnwlzc7bkpbj7s4d6limyt4f57t3cuqq5nuvhvwv2cu2a6iga2s64vjqcxjqiezyjooq, f1aszyvvnmz3cerzdz5ezoksl7odf7oaprdmb4oki
```

`miner-account` 关系表

初始 `miner-account`： 
- `f0128788-test`;

扩展后: 
- `f0128788-test`; 
- `f3wylwd6pclppme4qmbgwled5xpsbgwgqbn2alxa7yahg2gnbfkipsdv6m764xm5coizujmwdmkxeugplmorha-test`;
- `f3r47fkdzfmtex5ic3jnwlzc7bkpbj7s4d6limyt4f57t3cuqq5nuvhvwv2cu2a6iga2s64vjqcxjqiezyjooq-test`;
- `f1aszyvvnmz3cerzdz5ezoksl7odf7oaprdmb4oki-test`;
> owner 为 多签钱包时不支持一般的签名，故不做扩展。

（2） market作为客户端（client）时，发送订单的钱包地址不一定创建了矿工，所以须手动再映射个 `address(wallet)-account`

这种实现和最初的设计已经产生了背离，并引发了一些问题:

- 本地配置的和 `venus-auth` 获取的 `address-account` 混在一起，`venus-auth`中对miner的修改没法方便的更新到 `market` 的 `miner manager`;

- 某些场景下可能出现本地配置的 `account` 和 `venus-auth` 获取的重复，如：从`venus-auth` 获取 `f0128788-test`， 本地：`f3***-test`, 如果 `f3***` 与 `f0128788` 没有任何关联时将无法签名。

- 在目前market的部分功能，例如publish订单，会进行对相同地址的一些信息进行聚合，在进行签名的时候就无法区分是否是数据是来源于哪个account。（这个貌似没问题？）
