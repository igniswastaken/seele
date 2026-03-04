package sql

import (
	"fmt"
	"strings"

	"github.com/zerothy/seele/service"
)

type Engine struct {
	store *service.Store
}

func NewEngine(store *service.Store) *Engine {
	return &Engine{store: store}
}

func (e *Engine) Execute(query string) (interface{}, error) {
	lexer := NewLexer(query)
	parser := NewParser(lexer)

	stmt := parser.ParseStatement()

	if len(parser.Errors()) > 0 {
		return nil, fmt.Errorf("parser errors: %v", parser.Errors())
	}

	if stmt == nil {
		return nil, fmt.Errorf("failed to parse statement")
	}

	switch s := stmt.(type) {
	case *RevealStatement:
		return e.executeReveal(s)
	case *BanishStatement:
		return e.executeBanish(s)
	case *PlantStatement:
		return e.executePlant(s)
	case *MorphStatement:
		return e.executeMorph(s)
	default:
		return nil, fmt.Errorf("unsupported statement: %T", stmt)
	}
}

func (e *Engine) executeReveal(stmt *RevealStatement) (interface{}, error) {
	if stmt.IsPrefix {
		keys := e.store.Keys()
		var results []map[string]interface{}

		for _, k := range keys {
			if strings.HasPrefix(k, stmt.Prefix) {
				val, found := e.store.Get(k)
				if found {
					result := map[string]interface{}{
						"key":   k,
						"value": val,
					}
					results = append(results, result)
				}
			}
		}
		return results, nil
	}

	var results []map[string]interface{}

	for _, key := range stmt.Keys {
		val, found := e.store.Get(key)
		result := map[string]interface{}{
			"key": key,
		}
		if found {
			result["value"] = val
			result["found"] = true
		} else {
			result["found"] = false
		}
		results = append(results, result)
	}

	if len(stmt.Keys) == 1 {
		return results[0], nil
	}
	return results, nil
}

func (e *Engine) executeBanish(stmt *BanishStatement) (interface{}, error) {
	count := 0
	for _, key := range stmt.Keys {
		err := e.store.Delete(key)
		if err == nil {
			count++
		}
	}
	return fmt.Sprintf("Banished %d keys", count), nil
}

func (e *Engine) executePlant(stmt *PlantStatement) (interface{}, error) {
	count := 0
	for _, pair := range stmt.Pairs {
		set, err := e.store.SetIfNotExists(pair.Key, pair.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to plant %s: %v", pair.Key, err)
		}
		if !set {
			return nil, fmt.Errorf("key '%s' already exists — use MORPH to update it", pair.Key)
		}
		count++
	}
	return fmt.Sprintf("Planted %d keys", count), nil
}

func (e *Engine) executeMorph(stmt *MorphStatement) (interface{}, error) {
	count := 0
	for _, pair := range stmt.Pairs {
		set, err := e.store.SetIfExists(pair.Key, pair.Value)
		if err != nil {
			return nil, err
		}
		if !set {
			return nil, fmt.Errorf("cannot morph key %s: not found", pair.Key)
		}
		count++
	}
	return fmt.Sprintf("Morphed %d keys", count), nil
}
