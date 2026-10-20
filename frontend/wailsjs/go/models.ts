export namespace logging {
	
	export class Line {
	    level: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new Line(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	}

}

export namespace manager {
	
	export class ModuleInfo {
	    id: string;
	    name: string;
	    active: boolean;
	    usingEye: boolean;
	    usingExpression: boolean;
	    eyeActive: boolean;
	    expressionActive: boolean;
	    icons: string[];
	
	    static createFrom(source: any = {}) {
	        return new ModuleInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.active = source["active"];
	        this.usingEye = source["usingEye"];
	        this.usingExpression = source["usingExpression"];
	        this.eyeActive = source["eyeActive"];
	        this.expressionActive = source["expressionActive"];
	        this.icons = source["icons"];
	    }
	}
	export class Registered {
	    id: string;
	    name: string;
	    icons: string[];
	    supportsEye: boolean;
	    supportsExpression: boolean;
	    enabled: boolean;
	    useEye: boolean;
	    useExpression: boolean;
	    loaded: boolean;
	    active: boolean;
	    usingEye: boolean;
	    usingExpression: boolean;
	    eyeActive: boolean;
	    expressionActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Registered(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.icons = source["icons"];
	        this.supportsEye = source["supportsEye"];
	        this.supportsExpression = source["supportsExpression"];
	        this.enabled = source["enabled"];
	        this.useEye = source["useEye"];
	        this.useExpression = source["useExpression"];
	        this.loaded = source["loaded"];
	        this.active = source["active"];
	        this.usingEye = source["usingEye"];
	        this.usingExpression = source["usingExpression"];
	        this.eyeActive = source["eyeActive"];
	        this.expressionActive = source["expressionActive"];
	    }
	}

}

export namespace metadata {
	
	export class AppInfo {
	    description: string;
	    companyName: string;
	    productName: string;
	    productVersion: string;
	    copyright: string;
	    repository: string;
	    privacyUrl: string;
	    issuesUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.companyName = source["companyName"];
	        this.productName = source["productName"];
	        this.productVersion = source["productVersion"];
	        this.copyright = source["copyright"];
	        this.repository = source["repository"];
	        this.privacyUrl = source["privacyUrl"];
	        this.issuesUrl = source["issuesUrl"];
	    }
	}
	export class Author {
	    name: string;
	    email: string;
	
	    static createFrom(source: any = {}) {
	        return new Author(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.email = source["email"];
	    }
	}
	export class Metadata {
	    name: string;
	    author: Author;
	    info: AppInfo;
	
	    static createFrom(source: any = {}) {
	        return new Metadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.author = this.convertValues(source["author"], Author);
	        this.info = this.convertValues(source["info"], AppInfo);
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

export namespace mixer {
	
	export class ComponentState {
	    name: string;
	    kind: string;
	    min: number;
	    max: number;
	    value: any;
	
	    static createFrom(source: any = {}) {
	        return new ComponentState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.min = source["min"];
	        this.max = source["max"];
	        this.value = source["value"];
	    }
	}
	export class StageState {
	    name: string;
	    description: string;
	    active: boolean;
	    components: ComponentState[];
	
	    static createFrom(source: any = {}) {
	        return new StageState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.active = source["active"];
	        this.components = this.convertValues(source["components"], ComponentState);
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

export namespace osc {
	
	export class Target {
	    inPort: number;
	    outPort: number;
	    address: string;
	
	    static createFrom(source: any = {}) {
	        return new Target(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inPort = source["inPort"];
	        this.outPort = source["outPort"];
	        this.address = source["address"];
	    }
	}

}

export namespace oscquery {
	
	export class AvatarInfo {
	    name: string;
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new AvatarInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.id = source["id"];
	    }
	}

}

export namespace params {
	
	export class Definition {
	    Address: string;
	    Name: string;
	    Type: number;
	
	    static createFrom(source: any = {}) {
	        return new Definition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Address = source["Address"];
	        this.Name = source["Name"];
	        this.Type = source["Type"];
	    }
	}

}

export namespace sender {
	
	export class ParameterStats {
	    count: number;
	    deprecated: number;
	    hasNative: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ParameterStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.deprecated = source["deprecated"];
	        this.hasNative = source["hasNative"];
	    }
	}

}

export namespace service {
	
	export class OscStatus {
	    listenPort: number;
	    sendPort: number;
	    connected: boolean;
	    oscQuery: boolean;
	    messagesIn: number;
	    messagesOut: number;
	    allRelevant: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OscStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.listenPort = source["listenPort"];
	        this.sendPort = source["sendPort"];
	        this.connected = source["connected"];
	        this.oscQuery = source["oscQuery"];
	        this.messagesIn = source["messagesIn"];
	        this.messagesOut = source["messagesOut"];
	        this.allRelevant = source["allRelevant"];
	    }
	}

}

export namespace settings {
	
	export class Timing {
	    updateIntervalMs: number;
	    initTimeoutSec: number;
	    sendIntervalMs: number;
	
	    static createFrom(source: any = {}) {
	        return new Timing(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.updateIntervalMs = source["updateIntervalMs"];
	        this.initTimeoutSec = source["initTimeoutSec"];
	        this.sendIntervalMs = source["sendIntervalMs"];
	    }
	}

}

