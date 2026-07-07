package data

import (
	"context"
	"fmt"
	"time"

	"github.com/NeilXu2017/landau/log"
	"github.com/olivere/elastic/v7"
)

type (
	// ElasticSearchOptionFunc 参数设置
	ElasticSearchOptionFunc func(*ElasticSearchClient) error
	// ElasticSearchClient ES Client Wrapper
	ElasticSearchClient struct {
		Logger               string
		ServerURL            []string
		Sniff                bool
		Client               *elastic.Client
		ElasticServerVersion string
		RoutingKey           []string
		HighlightField       []string
		HealthCheck          bool
	}
	_ElasticSearchLogger struct {
		logger      string
		logCategory _ElasticSearchLoggerType
	}
	_ElasticSearchLoggerType string
	// ESSimpleQuery 简单类型的 Query
	ESSimpleQuery struct {
		Name       string
		Value      interface{}
		Type       int
		ClauseType int
		RangeType  int //Range Query
	}
)

const (
	//DefaultElasticSearchLogger default logger
	DefaultElasticSearchLogger                          = "main"
	_ElasticSearchErrorLogger  _ElasticSearchLoggerType = "error"
	_ElasticSearchInfoLogger   _ElasticSearchLoggerType = "info"
	_ElasticSearchTraceLogger  _ElasticSearchLoggerType = "debug"
	// ESTermQuery Term Query
	ESTermQuery = 1
	// ESMatchPhrase Phrase Query
	ESMatchPhrase = 2
	// ESFuzzyQuery Fuzzy Query
	ESFuzzyQuery = 3
	// ESMatchPhrasePrefix phrase prefix Query
	ESMatchPhrasePrefix = 4
	// ESTermsQuery Terms Query
	ESTermsQuery = 5
	// ESPrefix Prefix Query
	ESPrefix = 6
	// ESRange Range Query
	ESRange = 7
	// ESSearchBoolMust Must in Bool Query
	ESSearchBoolMust = 0
	// ESSearchBoolMustNot Must Not in Bool Query
	ESSearchBoolMustNot = 1
	// ESSearchBoolFilter Filter in Bool Query
	ESSearchBoolFilter = 2
	// ESSearchBoolShould Should in Bool Query
	ESSearchBoolShould = 3
)

// Printf implement logger interface
func (c *_ElasticSearchLogger) Printf(format string, v ...interface{}) {
	switch c.logCategory {
	case _ElasticSearchErrorLogger:
		log.Error2(c.logger, format, v...)
	case _ElasticSearchInfoLogger:
		log.Info2(c.logger, format, v...)
	case _ElasticSearchTraceLogger:
		log.Debug2(c.logger, format, v...)
	default:
		log.Info2(c.logger, format, v...)
	}
}

// SetElasticSearchURL 设置 Elasticsearch Server 地址
func SetElasticSearchURL(urls ...string) ElasticSearchOptionFunc {
	return func(c *ElasticSearchClient) error {
		switch len(urls) {
		case 0:
			c.ServerURL = []string{"http://127.0.0.1:9200"}
		default:
			c.ServerURL = urls
		}
		return nil
	}
}

// SetElasticSearchSniff 设置 Sniff
func SetElasticSearchSniff(sniff bool) ElasticSearchOptionFunc {
	return func(c *ElasticSearchClient) error {
		c.Sniff = sniff
		return nil
	}
}

// SetElasticSearchLogger 设置logger
func SetElasticSearchLogger(logger string) ElasticSearchOptionFunc {
	return func(c *ElasticSearchClient) error {
		c.Logger = logger
		return nil
	}
}

// SetElasticSearchRoutingKey 设置 RoutingKey
func SetElasticSearchRoutingKey(routingKey ...string) ElasticSearchOptionFunc {
	return func(c *ElasticSearchClient) error {
		c.RoutingKey = routingKey
		return nil
	}
}

// SetElasticSearchHighlightField 设置 HighlightField
func SetElasticSearchHighlightField(highlightField ...string) ElasticSearchOptionFunc {
	return func(c *ElasticSearchClient) error {
		c.HighlightField = highlightField
		return nil
	}
}

// SetElasticHealthCheck 设置 healthCheck
func SetElasticHealthCheck(healthCheck bool) ElasticSearchOptionFunc {
	return func(c *ElasticSearchClient) error {
		c.HealthCheck = healthCheck
		return nil
	}
}

// NewElasticSearchClient 构建ES Client
func NewElasticSearchClient(options ...ElasticSearchOptionFunc) (*ElasticSearchClient, error) {
	c := &ElasticSearchClient{
		Logger:         DefaultElasticSearchLogger,
		Sniff:          false,
		RoutingKey:     []string{},
		HighlightField: []string{},
		HealthCheck:    true,
	}
	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}
	errLogger := &_ElasticSearchLogger{logger: c.Logger, logCategory: _ElasticSearchErrorLogger}
	traceLogger := &_ElasticSearchLogger{logger: c.Logger, logCategory: _ElasticSearchTraceLogger}
	infoLogger := &_ElasticSearchLogger{logger: c.Logger, logCategory: _ElasticSearchInfoLogger}
	esClient, err := elastic.NewClient(elastic.SetURL(c.ServerURL...), elastic.SetSniff(c.Sniff), elastic.SetErrorLog(errLogger), elastic.SetTraceLog(traceLogger), elastic.SetInfoLog(infoLogger), elastic.SetHealthcheck(c.HealthCheck))
	if err != nil {
		log.Error2(c.Logger, "[ElasticSearch]\tNewClient url:%v error:%v", c.ServerURL, err)
		return nil, err
	}
	esVersion, err := esClient.ElasticsearchVersion(c.ServerURL[0])
	if err != nil {
		log.Error2(c.Logger, "[ElasticSearch]\tElasticsearchVersion url:%s error:%v", c.ServerURL[0], err)
		return nil, err
	}
	log.Info2(c.Logger, "[ElasticSearch] Server Version:%s", esVersion)
	c.Client = esClient
	return c, nil
}

// IndexExist 检测 index 是否存在
func (c *ElasticSearchClient) IndexExist(indexName string) (bool, error) {
	exists, err := c.Client.IndexExists(indexName).Do(context.Background())
	if err != nil {
		log.Error2(c.Logger, "[ElasticSearch]\tIndexExists error:%v", err)
	}
	return exists, err
}

// CreateIndex 创建 index
func (c *ElasticSearchClient) CreateIndex(indexName, mapping string) (*elastic.IndicesCreateResult, error) {
	indexResult, err := c.Client.CreateIndex(indexName).Body(mapping).Do(context.Background())
	if err != nil {
		log.Error2(c.Logger, "[ElasticSearch]\tCreateIndex error:%v indexName:%s mapping:%s", err, indexName, mapping)
	}
	return indexResult, err
}

// DeleteIndex  删除索引
func (c *ElasticSearchClient) DeleteIndex(indexName string) error {
	_, err := c.Client.DeleteIndex(indexName).Do(context.Background())
	return err
}

// UpdateIndexMapping 更新索引mapping定义
func (c *ElasticSearchClient) UpdateIndexMapping(indexName, mapping string) (*elastic.IndexResponse, error) {
	return c.Client.Index().Index(indexName).BodyString(mapping).Do(context.Background())
}

// PutRaw 存储数据
func (c *ElasticSearchClient) PutRaw(indexName, ID string, rawObject interface{}) (*elastic.IndexResponse, error) {
	p := c.Client.Index().Index(indexName)
	if ID != "" {
		p = p.Id(ID)
	}
	if len(c.RoutingKey) > 0 {
		p = p.Routing(c.RoutingKey[0])
	}
	indexResponse, err := p.BodyJson(rawObject).Do(context.Background())
	if err != nil {
		log.Error2(c.Logger, "[ElasticSearch]\tPutRaw error:%v indexName:%s ID:%s", err, indexName, ID)
	}
	return indexResponse, err
}

// PutString 存储数据
func (c *ElasticSearchClient) PutString(indexName, ID string, strJSON string) (*elastic.IndexResponse, error) {
	p := c.Client.Index().Index(indexName)
	if ID != "" {
		p = p.Id(ID)
	}
	if len(c.RoutingKey) > 0 {
		p = p.Routing(c.RoutingKey[0])
	}
	indexResponse, err := p.BodyString(strJSON).Do(context.Background())
	if err != nil {
		log.Error2(c.Logger, "[ElasticSearch]\tPutRaw error:%v indexName:%s ID:%s", err, indexName, ID)
	}
	return indexResponse, err
}

// UpdatePartial 更新部分数据
func (c *ElasticSearchClient) UpdatePartial(indexName, ID string, doc interface{}) (*elastic.UpdateResponse, int, error) {
	p := c.Client.Update().Index(indexName).Id(ID)
	if len(c.RoutingKey) > 0 {
		p = p.Routing(c.RoutingKey[0])
	}
	updateResponse, err := p.Doc(doc).Do(context.Background())
	if err != nil {
		if elastic.IsNotFound(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	return updateResponse, 1, nil
}

// GetByID 依据ID查找
func (c *ElasticSearchClient) GetByID(indexName, ID string) (*elastic.GetResult, int, error) {
	p := c.Client.Get().Index(indexName).Id(ID)
	if len(c.RoutingKey) > 0 {
		p = p.Routing(c.RoutingKey[0])
	}
	getResult, err := p.Do(context.Background())
	if err != nil {
		if elastic.IsNotFound(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	return getResult, 1, nil
}

// Search 查询
func (c *ElasticSearchClient) Search(indexName string, queries []ESSimpleQuery, offset, pageSize int, sort string, sortAsc bool) (*elastic.SearchResult, int, error) {
	if len(queries) == 0 {
		return nil, 0, fmt.Errorf("no query")
	}
	if offset < 0 || pageSize < 1 {
		return nil, 0, fmt.Errorf("offset or page size invalid")
	}
	start := time.Now()

	highlight := c.BuildHighlight()
	p := c.Client.Search(indexName).Routing(c.RoutingKey...).Highlight(highlight) // search 语句选项构建
	boolQuery := elastic.NewBoolQuery()                                           // bool query查询
	appendQuery := func(b *elastic.BoolQuery, query elastic.Query, boolType int) {
		switch boolType {
		case ESSearchBoolMust:
			b.Must(query)
		case ESSearchBoolMustNot:
			b.MustNot(query)
		case ESSearchBoolFilter:
			b.Filter(query)
		case ESSearchBoolShould:
			b.Should(query)
			b.MinimumNumberShouldMatch(1)
		}
	}
	for _, q := range queries {
		var query elastic.Query
		switch q.Type {
		case ESTermQuery:
			query = elastic.NewTermQuery(q.Name, q.Value)
		case ESTermsQuery:
			query = elastic.NewTermsQuery(q.Name, q.Value.([]interface{})...)
		case ESMatchPhrase:
			query = elastic.NewMatchPhraseQuery(q.Name, q.Value)
		case ESFuzzyQuery:
			query = elastic.NewFuzzyQuery(q.Name, q.Value)
		case ESMatchPhrasePrefix:
			query = elastic.NewMatchPhrasePrefixQuery(q.Name, q.Value)
		case ESPrefix:
			query = elastic.NewPrefixQuery(q.Name, q.Value.(string))
		case ESRange:
			rangeQuery := elastic.NewRangeQuery(q.Name)
			switch q.RangeType {
			case 1: //GT
				rangeQuery.Gt(q.Value)
			case 2: //GTE
				rangeQuery.Gte(q.Value)
			case 3: //LT
				rangeQuery.Lt(q.Value)
			case 4: //LTE
				rangeQuery.Lte(q.Value)
			}
			query = rangeQuery
		}
		appendQuery(boolQuery, query, q.ClauseType)
	}
	p = p.Query(boolQuery)
	if sort != "" {
		p = p.Sort(sort, sortAsc)
	}
	searchResult, err := p.From(offset).Size(pageSize).Pretty(true).Do(context.Background())
	if err != nil {
		if elastic.IsNotFound(err) {
			log.Info2(c.Logger, "[ElasticSearch]\tSearch\t[%s]\tIndex:%s\tQueries:%v\tResult:Not Found", time.Since(start), indexName, queries)
			return nil, 0, nil
		} else {
			log.Info2(c.Logger, "[ElasticSearch]\tSearch\t[%s]\tIndex:%s\tQueries:%v\tError:%s\n", time.Since(start), indexName, queries, err)
			return nil, 0, nil
		}
	}
	log.Info2(c.Logger, "[ElasticSearch]\tSearch\t[%s]\tIndex:%s\tQueries:%v\tResult:totalHits=%d", time.Since(start), indexName, queries, searchResult.TotalHits())
	return searchResult, int(searchResult.TotalHits()), nil
}

// BuildHighlight 构建返回*elastic.Highlight
func (c *ElasticSearchClient) BuildHighlight() *elastic.Highlight {
	fields := c.GetHighlightFields()
	h := elastic.NewHighlight().Fields(fields...)
	return h
}

// GetHighlightFields 获得HighlightFields
func (c *ElasticSearchClient) GetHighlightFields() []*elastic.HighlighterField {
	fields := make([]*elastic.HighlighterField, 0)
	for _, field := range c.HighlightField {
		fields = append(fields, elastic.NewHighlighterField(field))
	}
	return fields
}
