# 来源信息

- 仓库: Interview
- 文件: docs/Kaggle/competitions/getting-started/house-price/README.md
- 许可: CC BY-NC-SA 4.0

---

# **房价预测**

## 比赛说明

* [**房价预测**](https://www.kaggle.com/c/house-prices-advanced-regression-techniques)
* 要求购房者描述他们的梦想之家，他们可能不会从地下室天花板的高度或与东西方铁路的接近度开始。但是这个游乐场比赛的数据集证明，对价格谈判的影响远远超过卧室或白色栅栏的数量。
* 有79个解释变量描述（几乎）爱荷华州埃姆斯的住宅房屋的每个方面，这个竞赛挑战你预测每个房屋的最终价格。

## 参赛成员

* 开源组织: [ApacheCN ~ apachecn.org](http://www.apachecn.org/)

## 比赛分析

* 回归问题：价格的问题
* 常用算法： `回归`、`树回归`、`GBDT`、`xgboost`、`lightGBM`

```
步骤:
一. 数据分析
1. 下载并加载数据
2. 总体预览:了解每列数据的含义,数据的格式等
3. 数据初步分析,使用统计学与绘图:初步了解数据之间的相关性,为构造特征工程以及模型建立做准备

二. 特征工程
1.根据业务,常识,以及第二步的数据分析构造特征工程.
2.将特征转换为模型可以辨别的类型(如处理缺失值,处理文本进行等)

三. 模型选择
1.根据目标函数确定学习类型,是无监督学习还是监督学习,是分类问题还是回归问题等.
2.比较各个模型的分数,然后取效果较好的模型作为基础模型.

四. 模型融合
1. 可以参考泰坦尼克号的简单模型融合方式，通过对模型的对比打分方式选择合适的模型
2. 在房价预测里我们使用模型融合的方法来输出结果，最终的效果很好。

五. 修改特征和模型参数
1.可以通过添加或者修改特征,提高模型的上限.
2.通过修改模型的参数,是模型逼近上限
```

## 一. 数据分析

### 数据下载和加载

* 数据集下载地址：

```python
# 导入相关数据包
import numpy as np
import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
%matplotlib inline

from scipy import stats
from scipy.stats import norm
```

```python
root_path = '/opt/data/kaggle/getting-started/house-prices'

train = pd.read_csv('%s/%s' % (root_path, 'train.csv'))
test = pd.read_csv('%s/%s' % (root_path, 'test.csv'))
```

### 特征说明

```python
train.columns
```

    Index(['Id', 'MSSubClass', 'MSZoning', 'LotFrontage', 'LotArea', 'Street',
           'Alley', 'LotShape', 'LandContour', 'Utilities', 'LotConfig',
           'LandSlope', 'Neighborhood', 'Condition1', 'Condition2', 'BldgType',
           'HouseStyle', 'OverallQual', 'OverallCond', 'YearBuilt', 'YearRemodAdd',
           'RoofStyle', 'RoofMatl', 'Exterior1st', 'Exterior2nd', 'MasVnrType',
           'MasVnrArea', 'ExterQual', 'ExterCond', 'Foundation', 'BsmtQual',
           'BsmtCond', 'BsmtExposure', 'BsmtFinType1', 'BsmtFinSF1',
           'BsmtFinType2', 'BsmtFinSF2', 'BsmtUnfSF', 'TotalBsmtSF', 'Heating',
           'HeatingQC', 'CentralAir', 'Electrical', '1stFlrSF', '2ndFlrSF',
           'LowQualFinSF', 'GrLivArea', 'BsmtFullBath', 'BsmtHalfBath', 'FullBath',
           'HalfBath', 'BedroomAbvGr', 'KitchenAbvGr', 'KitchenQual',
           'TotRmsAbvGrd', 'Functional', 'Fireplaces', 'FireplaceQu', 'GarageType',
           'GarageYrBlt', 'GarageFinish', 'GarageCars', 'GarageArea', 'GarageQual',
           'GarageCond', 'PavedDrive', 'WoodDeckSF', 'OpenPorchSF',
           'EnclosedPorch', '3SsnPorch', 'ScreenPorch', 'PoolArea', 'PoolQC',
           'Fence', 'MiscFeature', 'MiscVal', 'MoSold', 'YrSold', 'SaleType',
           'SaleCondition', 'SalePrice'],
          dtype='object')

```python
train.info()
```

    RangeIndex: 1460 entries, 0 to 1459
    Data columns (total 81 columns):
    Id               1460 non-null int64
    MSSubClass       1460 non-null int64
    MSZoning         1460 non-null object
    LotFrontage      1201 non-null float64
    LotArea          1460 non-null int64
    Street           1460 non-null object
    Alley            91 non-null object
    LotShape         1460 non-null object
    LandContour      1460 non-null object
    Utilities        1460 non-null object
    LotConfig        1460 non-null object
    LandSlope        1460 non-null object
    Neighborhood     1460 non-null object
    Condition1       1460 non-null object
    Condition2       1460 non-null object
    BldgType         1460 non-null object
    HouseStyle       1460 non-null object
    OverallQual      1460 non-null int64
    OverallCond      1460 non-null int64
    YearBuilt        1460 non-null int64
    YearRemodAdd     1460 non-null int64
    RoofStyle        1460 non-null object
    RoofMatl         1460 non-null object
    Exterior1st      1460 non-null object
    Exterior2nd      1460 non-null object
    MasVnrType       1452 non-null object
    MasVnrArea       1452 non-null float64
    ExterQual        1460 non-null object
    ExterCond        1460 non-null object
    Foundation       1460 non-null object
    BsmtQual         1423 non-null object
    BsmtCond         1423 non-null object
    BsmtExposure     1422 non-null object
    BsmtFinType1     1423 non-null object
    BsmtFinSF1       1460 non-null int64
    BsmtFinType2     1422 non-null object
    BsmtFinSF2       1460 non-null int64
    BsmtUnfSF        1460 non-null int64
    TotalBsmtSF      1460 non-null int64
    Heating          1460 non-null object
    HeatingQC        1460 non-null object
    CentralAir       1460 non-null object
    Electrical       1459 non-null object
    1stFlrSF         1460 non-null int64
    2ndFlrSF         1460 non-null int64
    LowQualFinSF     1460 non-null int64
    GrLivArea        1460 non-null int64
    BsmtFullBath     1460 non-null int64
    BsmtHalfBath     1460 non-null int64
    FullBath         1460 non-null int64
    HalfBath         1460 non-null int64
    BedroomAbvGr     1460 non-null int64
    KitchenAbvGr     1460 non-null int64
    KitchenQual      1460 non-null object
    TotRmsAbvGrd     1460 non-null int64
    Functional       1460 non-null object
    Fireplaces       1460 non-null int64
    FireplaceQu      770 non-null object
    GarageType       1379 non-null object
    GarageYrBlt      1379 non-null float64
    GarageFinish     1379 non-null object
    GarageCars       1460 non-null int64
    GarageArea       1460 non-null int64
    GarageQual       1379 non-null object
    GarageCond       1379 non-null object
    PavedDrive       1460 non-null object
    WoodDeckSF       1460 non-null int64
    OpenPorchSF      1460 non-null int64
    EnclosedPorch    1460 non-null int64
    3SsnPorch        1460 non-null int64
    ScreenPorch      1460 non-null int64
    PoolArea         1460 non-null int64
    PoolQC           7 non-null object
    Fence            281 non-null object
    MiscFeature      54 non-null object
    MiscVal          1460 non-null int64
    MoSold           1460 non-null int64
    YrSold           1460 non-null int64
    SaleType         1460 non-null object
    SaleCondition    1460 non-null object
    SalePrice        1460 non-null int64
    dtypes: float64(3), int64(35), object(43)
    memory usage: 924.0+ KB

### 特征详情

```python
train.head(5)
```

      Id
      MSSubClass
      MSZoning
      LotFrontage
      LotArea
      Street
      Alley
      LotShape
      LandContour
      Utilities
      ...
      PoolArea
      PoolQC
      Fence
      MiscFeature
      MiscVal
      MoSold
      YrSold
      SaleType
      SaleCondition
      SalePrice
      0
      1
      2
      3
      4
5 rows × 81 columns

### 特征分析（统计学与绘图）

每一行是一条房子出售的记录，原始特征有80列，具体的意思可以根据data_description来查询，我们要预测的是房子的售价，即“SalePrice”。训练集有1459条记录，测试集有1460条记录，数据量还是很小的。

```python
# 相关性协方差表,corr()函数,返回结果接近0说明无相关性,大于0说明是正相关,小于0是负相关.
train_corr = train.drop('Id',axis=1).corr()
train_corr
```

      MSSubClass
      LotFrontage
      LotArea
      OverallQual
      OverallCond
      YearBuilt
      YearRemodAdd
      MasVnrArea
      BsmtFinSF1
      BsmtFinSF2
      ...
      WoodDeckSF
      OpenPorchSF
      EnclosedPorch
      3SsnPorch
      ScreenPorch
      PoolArea
      MiscVal
      MoSold
      YrSold
      SalePrice
      MSSubClass
      LotFrontage
      LotArea
      OverallQual
      OverallCond
      YearBuilt
      YearRemodAdd
      MasVnrArea
      BsmtFinSF1
      BsmtFinSF2
      BsmtUnfSF
      TotalBsmtSF
      1stFlrSF
      2ndFlrSF
      LowQualFinSF
      GrLivArea
      BsmtFullBath
      BsmtHalfBath
      FullBath
      HalfBath
      BedroomAbvGr
      KitchenAbvGr
      TotRmsAbvGrd
      Fireplaces
      GarageYrBlt
      GarageCars
      GarageArea
      WoodDeckSF
      OpenPorchSF
      EnclosedPorch
      3SsnPorch
      ScreenPorch
      PoolArea
      MiscVal
      MoSold
      YrSold
      SalePrice
37 rows × 37 columns

> 所有特征相关度分析

```python
# 画出相关性热力图
a = plt.subplots(figsize=(20, 12))#调整画布大小
a = sns.heatmap(train_corr, vmax=.8, square=True)#画热力图   annot=True 显示系数
```

> SalePrice 相关度特征排序

```python
# 寻找K个最相关的特征信息
k = 10 # number of variables for heatmap
cols = train_corr.nlargest(k, 'SalePrice')['SalePrice'].index
cm = np.corrcoef(train[cols].values.T)
sns.set(font_scale=1.5)
hm = plt.subplots(figsize=(20, 12))#调整画布大小
hm = sns.heatmap(cm, cbar=True, annot=True, square=True, fmt='.2f', annot_kws={'size': 10}, yticklabels=cols.values, xticklabels=cols.values)
plt.show()

'''
1. GarageCars 和 GarageAre 相关性很高、就像双胞胎一样，所以我们只需要其中的一个变量，例如：GarageCars。
2. TotalBsmtSF  和 1stFloor 与上述情况相同，我们选择 TotalBsmtS
3. GarageAre 和 TotRmsAbvGrd 与上述情况相同，我们选择 GarageAre
'''
```

    '\n1. GarageCars 和 GarageAre 相关性很高、就像双胞胎一样，所以我们只需要其中的一个变量，例如：GarageCars。\n2. TotalBsmtSF  和 1stFloor 与上述情况相同，我们选择 TotalBsmtS\n3. GarageAre 和 TotRmsAbvGrd 与上述情况相同，我们选择 GarageAre\n'

> SalePrice 和相关变量之间的散点图

```python
sns.set()
cols = ['SalePrice', 'OverallQual', 'GrLivArea','GarageCars', 'TotalBsmtSF', 'FullBath', 'YearBuilt']
sns.pairplot(train[cols], size = 2.5)
plt.show();
```

```python
train[['SalePrice', 'OverallQual', 'GrLivArea','GarageCars', 'TotalBsmtSF', 'FullBath', 'YearBuilt']].info()
```

    RangeIndex: 1460 entries, 0 to 1459
    Data columns (total 7 columns):
    SalePrice      1460 non-null int64
    OverallQual    1460 non-null int64
    GrLivArea      1460 non-null int64
    GarageCars     1460 non-null int64
    TotalBsmtSF    1460 non-null int64
    FullBath       1460 non-null int64
    YearBuilt      1460 non-null int64
    dtypes: int64(7)
    memory usage: 79.9 KB

## 二. 特征工程

```
test['SalePrice'] = None
train_test = pd.concat((train, test)).reset_index(drop=True)
```

### 1. 缺失值分析

2. 根据业务,常识,以及第二步的数据分析构造特征工程.
2. 将特征转换为模型可以辨别的类型(如处理缺失值,处理文本进行等)

```python
total= train_test.isnull().sum().sort_values(ascending=False)
percent = (train_test.isnull().sum()/train_test.isnull().count()).sort_values(ascending=False)
missing_data = pd.concat([total, percent], axis=1, keys=['Total','Lost Percent'])

print(missing_data[missing_data.isnull().values==False].sort_values('Total', axis=0, ascending=False).head(20))

'''
1. 对于缺失率过高的特征，例如 超过15% 我们应该删掉相关变量且假设该变量并不存在
2. GarageX 变量群的缺失数据量和概率都相同，可以选择一个就行，例如：GarageCars
3. 对于缺失数据在5%左右（缺失率低），可以直接删除/回归预测
'''
```

    '\n1. 对于缺失率过高的特征，例如 超过15% 我们应该删掉相关变量且假设该变量并不存在\n2. GarageX 变量群的缺失数据量和概率都相同，可以选择一个就行，例如：GarageCars\n3. 对于缺失数据在5%左右（缺失率低），可以直接删除/回归预测\n'

```python
train_test = train_test.drop((missing_data[missing_data['Total'] > 1]).index.drop('SalePrice') , axis=1)
# train_test = train_test.drop(train.loc[train['Electrical'].isnull()].index)

tmp = train_test[train_test['SalePrice'].isnull().values==False]
print(tmp.isnull().sum().max()) # justchecking that there's no missing data missing
```

    1

### 2. 异常值处理

#### 单因素分析

这里的关键在于如何建立阈值，定义一个观察值为异常值。我们对数据进行正态化，意味着把数据值转换成均值为 0，方差为 1 的数据

```python
fig = plt.figure(figsize=(12, 6))
ax1 = fig.add_subplot(1, 2, 1)
ax2 = fig.add_subplot(1, 2, 2)
ax1.hist(train.SalePrice)
ax2.hist(np.log1p(train.SalePrice))

'''
从直方图中可以看出：

* 偏离正态分布
* 数据正偏
* 有峰值
'''
# 数据偏度和峰度度量：

print("Skewness: %f" % train['SalePrice'].skew())
print("Kurtosis: %f" % train['SalePrice'].kurt())

'''
低范围的值都比较相似并且在 0 附近分布。
高范围的值离 0 很远，并且七点几的值远在正常范围之外。
'''
```

    '\n低范围的值都比较相似并且在 0 附近分布。\n高范围的值离 0 很远，并且七点几的值远在正常范围之外。\n'

#### 双变量分析

> 1.GrLivArea 和 SalePrice 双变量分析

```python
var = 'GrLivArea'
data = pd.concat([train['SalePrice'], train[var]], axis=1)
data.plot.scatter(x=var, y='SalePrice', ylim=(0,800000));

'''
从图中可以看出：

1. 有两个离群的 GrLivArea 值很高的数据，我们可以推测出现这种情况的原因。
    或许他们代表了农业地区，也就解释了低价。 这两个点很明显不能代表典型样例，所以我们将它们定义为异常值并删除。
2. 图中顶部的两个点是七点几的观测值，他们虽然看起来像特殊情况，但是他们依然符合整体趋势，所以我们将其保留下来。
'''
```

    '\n从图中可以看出：\n\n1. 有两个离群的 GrLivArea 值很高的数据，我们可以推测出现这种情况的原因。\n    或许他们代表了农业地区，也就解释了低价。 这两个点很明显不能代表典型样例，所以我们将它们定义为异常值并删除。\n2. 图中顶部的两个点是七点几的观测值，他们虽然看起来像特殊情况，但是他们依然符合整体趋势，所以我们将其保留下来。\n'

```python
# 删除点
print(train.sort_values(by='GrLivArea', ascending = False)[:2])
tmp = train_test[train_test['SalePrice'].isnull().values==False]

train_test = train_test.drop(tmp[tmp['Id'] == 1299].index)
train_test = train_test.drop(tmp[tmp['Id'] == 524].index)
```

> 2.TotalBsmtSF 和 SalePrice 双变量分析

```python
var = 'TotalBsmtSF'
data = pd.concat([train['SalePrice'],train[var]], axis=1)
data.plot.scatter(x=var, y='SalePrice',ylim=(0,800000))
```

### 核心部分

“房价” 到底是谁？

这个问题的答案，需要我们验证根据数据基础进行多元分析的假设。

我们已经进行了数据清洗，并且发现了 SalePrice 的很多信息，现在我们要更进一步理解 SalePrice 如何遵循统计假设，可以让我们应用多元技术。

应该测量 4 个假设量：

* 正态性
* 同方差性
* 线性
* 相关错误缺失

#### 正态性：

应主要关注以下两点：直方图 – 峰度和偏度。

正态概率图 – 数据分布应紧密跟随代表正态分布的对角线。

1.  SalePrice 绘制直方图和正态概率图：

```python
sns.distplot(train['SalePrice'], fit=norm)
fig = plt.figure()
res = stats.probplot(train['SalePrice'], plot=plt)

'''
可以看出，房价分布不是正态的，显示了峰值，正偏度，但是并不跟随对角线。
可以用对数变换来解决这个问题
'''
```

    '\n可以看出，房价分布不是正态的，显示了峰值，正偏度，但是并不跟随对角线。\n可以用对数变换来解决这个问题\n'

```python
# 进行对数变换：
# 进行对数变换：
train_test['SalePrice'] = [i if i is None else np.log1p(i) for i in train_test['SalePrice']]
```

```python
# 绘制变换后的直方图和正态概率图：
tmp = train_test[train_test['SalePrice'].isnull().values==False]

sns.distplot(tmp[tmp['SalePrice'] !=0]['SalePrice'], fit=norm);
fig = plt.figure()
res = stats.probplot(tmp['SalePrice'], plot=plt)
```

#### 2. GrLivArea
绘制直方图和正态概率曲线图：

```python
sns.distplot(train['GrLivArea'], fit=norm);
fig = plt.figure()
res = stats.probplot(train['GrLivArea'], plot=plt)
```

```python
# 进行对数变换：
train_test['GrLivArea'] = [i if i is None else np.log1p(i) for i in train_test['GrLivArea']]

# 绘制变换后的直方图和正态概率图：
tmp = train_test[train_test['SalePrice'].isnull().values==False]
sns.distplot(tmp['GrLivArea'], fit=norm)
fig = plt.figure()
res = stats.probplot(tmp['GrLivArea'], plot=plt)
```

#### 3.TotalBsmtSF

绘制直方图和正态概率曲线图：

```python
sns.distplot(train['TotalBsmtSF'],fit=norm);
fig = plt.figure()
res = stats.probplot(train['TotalBsmtSF'],plot=plt)

'''
从图中可以看出：
* 显示出了偏度
* 大量为 0(Y值) 的观察值（没有地下室的房屋）
* 含 0(Y值) 的数据无法进行对数变换
'''
```

    '\n从图中可以看出：\n* 显示出了偏度\n* 大量为 0(Y值) 的观察值（没有地下室的房屋）\n* 含 0(Y值) 的数据无法进行对数变换\n'

```python
# 去掉为0的分布情况
tmp = train_test[train_test['SalePrice'].isnull().values==False]

tmp = np.array(tmp.loc[tmp['TotalBsmtSF']>0, ['TotalBsmtSF']])[:, 0]
sns.distplot(tmp, fit=norm)
fig = plt.figure()
res = stats.probplot(tmp, plot=plt)
```

```python
# 我们建立了一个变量，可以得到有没有地下室的影响值（二值变量），我们选择忽略零值，只对非零值进行对数变换。
# 这样我们既可以变换数据，也不会损失有没有地下室的影响。

print(train.loc[train['TotalBsmtSF']==0, ['TotalBsmtSF']].count())
train.loc[train['TotalBsmtSF']==0,'TotalBsmtSF'] = 1
print(train.loc[train['TotalBsmtSF']==1, ['TotalBsmtSF']].count())
```

    TotalBsmtSF    37
    dtype: int64
    TotalBsmtSF    37
    dtype: int64

```python
# 进行对数变换：
tmp = train_test[train_test['SalePrice'].isnull().values==False]

print(tmp['TotalBsmtSF'].head(10))
train_test['TotalBsmtSF']= np.log1p(train_test['TotalBsmtSF'])

tmp = train_test[train_test['SalePrice'].isnull().values==False]
print(tmp['TotalBsmtSF'].head(10))
```

    0     856.0
    1    1262.0
    2     920.0
    3     756.0
    4    1145.0
    5     796.0
    6    1686.0
    7    1107.0
    8     952.0
    9     991.0
    Name: TotalBsmtSF, dtype: float64
    0    6.753438
    1    7.141245
    2    6.825460
    3    6.629363
    4    7.044033
    5    6.680855
    6    7.430707
    7    7.010312
    8    6.859615
    9    6.899723
    Name: TotalBsmtSF, dtype: float64

```python
# 绘制变换后的直方图和正态概率图：
tmp = train_test[train_test['SalePrice'].isnull().values==False]

tmp = np.array(tmp.loc[tmp['TotalBsmtSF']>0, ['TotalBsmtSF']])[:, 0]
sns.distplot(tmp, fit=norm)
fig = plt.figure()
res = stats.probplot(tmp, plot=plt)
```

#### 同方差性：

最好的测量两个变量的同方差性的方法就是图像。

1.  SalePrice 和 GrLivArea 同方差性

绘制散点图：

```python
tmp = train_test[train_test['SalePrice'].isnull().values==False]

plt.scatter(tmp['GrLivArea'], tmp['SalePrice'])
```

2. SalePrice with TotalBsmtSF 同方差性

绘制散点图：

```python
tmp = train_test[train_test['SalePrice'].isnull().values==False]

plt.scatter(tmp[tmp['TotalBsmtSF']>0]['TotalBsmtSF'], tmp[tmp['TotalBsmtSF']>0]['SalePrice'])

# 可以看出 SalePrice 在整个 TotalBsmtSF 变量范围内显示出了同等级别的变化。
```

## 三. 模型选择

### 1.数据标准化

```python
tmp = train_test[train_test['SalePrice'].isnull().values==False]
tmp_1 = train_test[train_test['SalePrice'].isnull().values==True]

x_train = tmp[['OverallQual', 'GrLivArea','GarageCars', 'TotalBsmtSF', 'FullBath', 'YearBuilt']]
y_train = tmp[["SalePrice"]].values.ravel()
x_test = tmp_1[['OverallQual', 'GrLivArea','GarageCars', 'TotalBsmtSF', 'FullBath', 'YearBuilt']]

# 简单测试，用中位数来替代
# print(x_test.GarageCars.mean(), x_test.GarageCars.median(), x_test.TotalBsmtSF.mean(), x_test.TotalBsmtSF.median())

x_test["GarageCars"].fillna(x_test.GarageCars.median(), inplace=True)
x_test["TotalBsmtSF"].fillna(x_test.TotalBsmtSF.median(), inplace=True)
```

### 2.开始建模

1. 可选单个模型模型有 线性回归（Ridge、Lasso）、树回归、GBDT、XGBoost、LightGBM 等.
2. 也可以将多个模型组合起来,进行模型融合,比如voting,stacking等方法
3. 好的特征决定模型上限,好的模型和参数可以无线逼近上限.
4. 我测试了多种模型,模型结果最高的随机森林,最高有0.8.

#### bagging:

单个分类器的效果真的是很有限。
我们会倾向于把N多的分类器合在一起，做一个“综合分类器”以达到最好的效果。
我们从刚刚的试验中得知，Ridge(alpha=15)给了我们最好的结果。

```python
from sklearn.linear_model import Ridge
from sklearn.model_selection import cross_val_score
from sklearn.ensemble import BaggingRegressor, RandomForestRegressor

ridge = Ridge(alpha=0.1)

# bagging 把很多小的分类器放在一起，每个train随机的一部分数据，然后把它们的最终结果综合起来（多数投票）
# bagging 算是一种算法框架
params = [1, 10, 20, 40, 60]
test_scores = []
for param in params:
    clf = BaggingRegressor(base_estimator=ridge, n_estimators=param)
    # cv=5表示cross_val_score采用的是k-fold cross validation的方法，重复5次交叉验证
    # scoring='precision'、scoring='recall'、scoring='f1', scoring='neg_mean_squared_error' 方差值
    test_score = np.sqrt(-cross_val_score(clf, x_train, y_train, cv=10, scoring='neg_mean_squared_error'))
    test_scores.append(np.mean(test_score))

print(test_score.mean())
plt.plot(params, test_scores)
plt.title('n_estimators vs CV Error')
plt.show()
```

```python
# 模型选择
## LASSO Regression :
lasso = make_pipeline(RobustScaler(), Lasso(alpha=0.0005, random_state=1))
* Elastic Net Regression
ENet = make_pipeline(
    RobustScaler(), ElasticNet(
        alpha=0.0005, l1_ratio=.9, random_state=3))
Kernel Ridge Regression
KRR = KernelRidge(alpha=0.6, kernel='polynomial', degree=2, coef0=2.5)
## Gradient Boosting Regression
GBoost = GradientBoostingRegressor(
    n_estimators=3000,
    learning_rate=0.05,
    max_depth=4,
    max_features='sqrt',
    min_samples_leaf=15,
    min_samples_split=10,
    loss='huber',
    random_state=5)
## XGboost
model_xgb = xgb.XGBRegressor(
    colsample_bytree=0.4603,
    gamma=0.0468,
    learning_rate=0.05,
    max_depth=3,
    min_child_weight=1.7817,
    n_estimators=2200,
    reg_alpha=0.4640,
    reg_lambda=0.8571,
    subsample=0.5213,
    silent=1,
    random_state=7,
    nthread=-1)
## lightGBM
model_lgb = lgb.LGBMRegressor(
    objective='regression',
    num_leaves=5,
    learning_rate=0.05,
    n_estimators=720,
    max_bin=55,
    bagging_fraction=0.8,
    bagging_freq=5,
    feature_fraction=0.2319,
    feature_fraction_seed=9,
    bagging_seed=9,
    min_data_in_leaf=6,
    min_sum_hessian_in_leaf=11)
## 对这些基本模型进行打分
score = rmsle_cv(lasso)
print("\nLasso score: {:.4f} ({:.4f})\n".format(score.mean(), score.std()))
score = rmsle_cv(ENet)
print("ElasticNet score: {:.4f} ({:.4f})\n".format(score.mean(), score.std()))
score = rmsle_cv(KRR)
print(
    "Kernel Ridge score: {:.4f} ({:.4f})\n".format(score.mean(), score.std()))
score = rmsle_cv(GBoost)
print("Gradient Boosting score: {:.4f} ({:.4f})\n".format(score.mean(),
                                                          score.std()))
score = rmsle_cv(model_xgb)
print("Xgboost score: {:.4f} ({:.4f})\n".format(score.mean(), score.std()))
score = rmsle_cv(model_lgb)
print("LGBM score: {:.4f} ({:.4f})\n".format(score.mean(), score.std()))
```

```python
from sklearn.linear_model import Ridge
from sklearn.model_selection import learning_curve

ridge = Ridge(alpha=0.1)

train_sizes, train_loss, test_loss = learning_curve(ridge, x_train, y_train, cv=10,
                                                    scoring='neg_mean_squared_error',
                                                    train_sizes = [0.1, 0.3, 0.5, 0.7, 0.9 , 0.95, 1])

# 训练误差均值
train_loss_mean = -np.mean(train_loss, axis = 1)
# 测试误差均值
test_loss_mean = -np.mean(test_loss, axis = 1)

# 绘制误差曲线
plt.plot(train_sizes/len(x_train), train_loss_mean, 'o-', color = 'r', label = 'Training')
plt.plot(train_sizes/len(x_train), test_loss_mean, 'o-', color = 'g', label = 'Cross-Validation')

plt.xlabel('Training data size')
plt.ylabel('Loss')
plt.legend(loc = 'best')
plt.show()
```

```python
mode_br = BaggingRegressor(base_estimator=ridge, n_estimators=10)
mode_br.fit(x_train, y_train)
y_test = np.expm1(mode_br.predict(x_test))
```

## 四 建立模型

> 模型融合 voting

```python
# 模型融合
class AveragingModels(BaseEstimator, RegressorMixin, TransformerMixin):
    def __init__(self, models):
        self.models = models

    # we define clones of the original models to fit the data in
    def fit(self, X, y):
        self.models_ = [clone(x) for x in self.models]

        # Train cloned base models
        for model in self.models_:
            model.fit(X, y)

        return self

    # Now we do the predictions for cloned models and average them
    def predict(self, X):
        predictions = np.column_stack(
            [model.predict(X) for model in self.models_])
        return np.mean(predictions, axis=1)

# 评价这四个模型的好坏
averaged_models = AveragingModels(models=(ENet, GBoost, KRR, lasso))
score = rmsle_cv(averaged_models)
print(" Averaged base models score: {:.4f} ({:.4f})\n".format(score.mean(),
                                                              score.std()))

# 最终对模型的训练和预测
# StackedRegressor
stacked_averaged_models.fit(train.values, y_train)
stacked_train_pred = stacked_averaged_models.predict(train.values)
stacked_pred = np.expm1(stacked_averaged_models.predict(test.values))
print(rmsle(y_train, stacked_train_pred))

# XGBoost
model_xgb.fit(train, y_train)
xgb_train_pred = model_xgb.predict(train)
xgb_pred = np.expm1(model_xgb.predict(test))
print(rmsle(y_train, xgb_train_pred))
# lightGBM
model_lgb.fit(train, y_train)
lgb_train_pred = model_lgb.predict(train)
lgb_pred = np.expm1(model_lgb.predict(test.values))
print(rmsle(y_train, lgb_train_pred))
'''RMSE on the entire Train data when averaging'''

print('RMSLE score on train data:')
print(rmsle(y_train, stacked_train_pred * 0.70 + xgb_train_pred * 0.15 +
            lgb_train_pred * 0.15))
# 模型融合的预测效果
ensemble = stacked_pred * 0.70 + xgb_pred * 0.15 + lgb_pred * 0.15
# 保存结果
result = pd.DataFrame()
result['Id'] = test_ID
result['SalePrice'] = ensemble
# index=False 是用来除去行编号
result.to_csv('/Users/liudong/Desktop/house_price/result.csv', index=False)
```
