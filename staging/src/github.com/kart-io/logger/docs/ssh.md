# 使用多个 SSH Key 管理 GitHub 账号

## 在日常开发中，我们可能会遇到以下场景

- 个人 GitHub 账号 和 公司 GitHub 账号 共存
- 不同的项目需要使用 不同的 SSH Key
- 同一台电脑上需要区分多个身份（比如开源贡献和工作项目）

如果只用一个 ~/.ssh/id_rsa，就会导致身份混乱甚至 push 失败。本文将介绍如何通过 配置 SSH Config 文件，轻松管理多个 GitHub 账号。

## 1. 生成不同的 SSH Key

```sh
# 个人账号
ssh-keygen -t rsa -b 4096 -C "your_personal_email@example.com" -f ~/.ssh/id_rsa_github

# 公司账号
ssh-keygen -t rsa -b 4096 -C "your_work_email@example.com" -f ~/.ssh/id_rsa

```

执行后会在 ~/.ssh/ 下生成对应的私钥和公钥文件：

- id_rsa / id_rsa.pub → 公司账号
- id_rsa_github / id_rsa_github.pub → 个人账号

将 .pub 公钥文件添加到对应 GitHub 账号的 SSH Keys 设置里。

## 2. 配置 ~/.ssh/config

```sh
# 公司账号（kart）
Host kart.github.com
HostName github.com
User git
PreferredAuthentications publickey
IdentityFile ~/.ssh/id_rsa
#指定特定的ssh私钥文件

Host github.com
HostName github.com
User git
PreferredAuthentications publickey
IdentityFile ~/.ssh/id_rsa_github
# 指定特定的ssh私钥文件
```

这里的关键点是：

- Host kart.github.com → 给公司账号起了个别名 kart.github.com
- Host github.com → 默认个人账号

这样就可以在同一台机器上使用两个不同的 SSH Key。

## 3. 使用方法

- 克隆公司项目 时，使用别名：

```sh
git clone git@kart.github.com:CompanyOrg/Repo.git
```

- 克隆个人项目 时，使用默认 GitHub 域名：

```sh
git clone git@github.com:YourName/Repo.git
```

## 4. 验证配置

可以通过 ssh -T 来验证是否生效：

```sh
ssh -T git@github.com
# Hi YourName! You've successfully authenticated...

ssh -T git@kart.github.com
# Hi CompanyUser! You've successfully authenticated...
```
