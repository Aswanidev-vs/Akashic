export namespace main {
	
	export class AISettings {
	    enabled: boolean;
	    endpoint: string;
	    defaultModel: string;
	    temperature: number;
	    maxTokens: number;
	    availableModels: string[];
	
	    static createFrom(source: any = {}) {
	        return new AISettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.endpoint = source["endpoint"];
	        this.defaultModel = source["defaultModel"];
	        this.temperature = source["temperature"];
	        this.maxTokens = source["maxTokens"];
	        this.availableModels = source["availableModels"];
	    }
	}
	export class EditorEventData {
	    filePath: string;
	    selection?: string;
	    cursorLine: number;
	    cursorCol: number;
	
	    static createFrom(source: any = {}) {
	        return new EditorEventData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.selection = source["selection"];
	        this.cursorLine = source["cursorLine"];
	        this.cursorCol = source["cursorCol"];
	    }
	}
	export class EditorSettings {
	    fontFamily: string;
	    fontSize: number;
	    lineHeight: number;
	    tabSize: number;
	    useSpaces: boolean;
	    wordWrap: boolean;
	    lineNumbers: boolean;
	    minimap: boolean;
	    autoSave: boolean;
	    autoSaveDelay: number;
	    showWhitespace: boolean;
	    highlightActiveLine: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EditorSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fontFamily = source["fontFamily"];
	        this.fontSize = source["fontSize"];
	        this.lineHeight = source["lineHeight"];
	        this.tabSize = source["tabSize"];
	        this.useSpaces = source["useSpaces"];
	        this.wordWrap = source["wordWrap"];
	        this.lineNumbers = source["lineNumbers"];
	        this.minimap = source["minimap"];
	        this.autoSave = source["autoSave"];
	        this.autoSaveDelay = source["autoSaveDelay"];
	        this.showWhitespace = source["showWhitespace"];
	        this.highlightActiveLine = source["highlightActiveLine"];
	    }
	}
	export class FileInfo {
	    path: string;
	    name: string;
	    encoding: string;
	    lineEnding: string;
	    isDirty: boolean;
	    isNewFile: boolean;
	    lastSaved: number;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.encoding = source["encoding"];
	        this.lineEnding = source["lineEnding"];
	        this.isDirty = source["isDirty"];
	        this.isNewFile = source["isNewFile"];
	        this.lastSaved = source["lastSaved"];
	    }
	}
	export class FileOpenResult {
	    fileInfo?: FileInfo;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new FileOpenResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileInfo = this.convertValues(source["fileInfo"], FileInfo);
	        this.content = source["content"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OllamaModel {
	    name: string;
	    size: string;
	    modified: string;
	    parameters: string;
	
	    static createFrom(source: any = {}) {
	        return new OllamaModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.size = source["size"];
	        this.modified = source["modified"];
	        this.parameters = source["parameters"];
	    }
	}
	export class OllamaStatus {
	    installed: boolean;
	    version: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new OllamaStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.version = source["version"];
	        this.message = source["message"];
	    }
	}
	export class UISettings {
	    theme: string;
	    darkMode: boolean;
	    showStatusBar: boolean;
	    compactMode: boolean;
	    sidebarPosition: string;
	
	    static createFrom(source: any = {}) {
	        return new UISettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.darkMode = source["darkMode"];
	        this.showStatusBar = source["showStatusBar"];
	        this.compactMode = source["compactMode"];
	        this.sidebarPosition = source["sidebarPosition"];
	    }
	}
	export class Settings {
	    editor: EditorSettings;
	    ui: UISettings;
	    ai: AISettings;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.editor = this.convertValues(source["editor"], EditorSettings);
	        this.ui = this.convertValues(source["ui"], UISettings);
	        this.ai = this.convertValues(source["ai"], AISettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

